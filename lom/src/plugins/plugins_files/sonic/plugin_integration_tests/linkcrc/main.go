package main

import (
    "context"
    "github.com/go-redis/redis"
    "lom/src/lib/lomcommon"
    "lom/src/lib/lomipc"
    "lom/src/plugins/plugins_common"
    "lom/src/plugins/sonic/plugin/linkcrc"
    "lom/src/plugins/sonic/plugin_integration_tests/linkcrc_mocker/linkcrc_utils"
    "lom/src/plugins/sonic/plugin_integration_tests/utils"
    "os"
    "os/exec"
    "strings"
    "time"
)

const (
    redis_address                    = "localhost:6379"
    redis_counters_db                = 2
    redis_app_db                     = 0
    fileName                         = "./LINK_CRC_COUNTERS"
    redis_password                   = ""
    counter_poll_disable_command     = "sudo counterpoll port disable"
    action_name                      = "link_crc_detection"
    detection_type                   = "detection"
    counter_poll_enable_command      = "sudo counterpoll port enable"
    counters_port_name_map_redis_key = "COUNTERS_PORT_NAME_MAP"
    admin_status                     = "admin_status"
    oper_status                      = "oper_status"
    ifUp                             = "up"
    counters_db                      = "COUNTERS:"
)

func main() {
    // Pre - setup
    utils.PrintInfo("Starting Link CRC Detection plugin integration test.")
    _, err := exec.Command("/bin/sh", "-c", counter_poll_disable_command).Output()
    if err != nil {
        utils.PrintError("Error disabling counterpoll on switch %v", err)
    } else {
        utils.PrintInfo("Successfuly Disabled counterpoll")
    }

    outliersArray := []string{"1,0,0,0", "1,0,1,0,0", "1,0,0,0,0"}
    shouldDetectCrc := []bool{true, true, false}

    for index := 0; index < len(outliersArray); index++ {
        ctx, cancelFunc := context.WithCancel(context.Background())
        shouldDetect := shouldDetectCrc[index]
        go InvokeLinkCrcDetectionPlugin(outliersArray[index], ctx, cancelFunc, shouldDetect)
        timeoutTimer := time.NewTimer(time.Duration(3) * time.Minute)
    loop:
        for {
            select {
            case <-timeoutTimer.C:
                if shouldDetect {
                    utils.PrintError("Timeout. Aborting Integration test")
                } else {
                    utils.PrintInfo("Integration Test Succeeded as timeout was expected")
                }
                break loop
            case <-ctx.Done():
                if !shouldDetect {
                    utils.PrintError("Integration Test Failed")
                }
                break loop
            }
        }
        timeoutTimer.Stop()
    }

    // Post - clean up
    _, err = exec.Command("/bin/sh", "-c", counter_poll_enable_command).Output()
    if err != nil {
        utils.PrintError("Error enabling counterpoll on switch %v", err)
    } else {
        utils.PrintInfo("Successfuly Enabled counterpoll")
    }
    utils.PrintInfo("Its exepcted not to receive any heartbeat or plugin logs from now as the anomaly is detected")
}

func InvokeLinkCrcDetectionPlugin(pattern string, ctx context.Context, cancelFunc context.CancelFunc, shouldDetect bool) {
    go linkcrc_utils.MockRedisWithLinkCrcCounters(pattern, 10, os.Args[1:], ctx)
    linkCrcDetectionPlugin := linkcrc.LinkCRCDetectionPlugin{}
    actionCfg := lomcommon.ActionCfg_t{Name: action_name, Type: detection_type, Timeout: 0, HeartbeatInt: 10, Disable: false, Mimic: false, ActionKnobs: ""}
    linkCrcDetectionPlugin.Init(&actionCfg)
    actionRequest := lomipc.ActionRequestData{Action: action_name, InstanceId: "InstId", AnomalyInstanceId: "AnInstId", AnomalyKey: "", Timeout: 0}
    pluginHBChan := make(chan plugins_common.PluginHeartBeat, 10)
    go utils.ReceiveAndLogHeartBeat(pluginHBChan)
    time.Sleep(10 * time.Second)
    response := linkCrcDetectionPlugin.Request(pluginHBChan, &actionRequest)
    utils.PrintInfo("Integration testing Done.Anomaly detection result: %s", response.AnomalyKey)
    if shouldDetect {
        for _, v := range os.Args[1:] {
            if !strings.Contains(response.AnomalyKey, v) {
                utils.PrintError("Integration Test Failed")
                cancelFunc()
                return
            }
        }
        utils.PrintInfo("Integration Test Succeeded")
    }
    cancelFunc()
}

func MockRedisData(outliers []int) error {
    var countersDbClient = redis.NewClient(&redis.Options{
        Addr:     redis_address,
        Password: redis_password,
        DB:       redis_counters_db,
    })

    var appDbClient = redis.NewClient(&redis.Options{
        Addr:     redis_address,
        Password: redis_password,
        DB:       redis_app_db,
    })

    // Get all ifName to oid mappings.
    interfaceToOidMapping, err := countersDbClient.HGetAll(counters_port_name_map_redis_key).Result()
    if err != nil {
        utils.PrintError("Error fetching counters port name map %v", err)
        return err
    }

    // Filter only those required as per the links passed through arguments
    mockedLinks := make(map[string]string)
    for _, v := range os.Args[1:] {
        mockedLinks[v] = interfaceToOidMapping[v]
    }

    // Set admin and oper status as up for mocked links.
    ifStatusMock := map[string]interface{}{admin_status: ifUp, oper_status: ifUp}
    for k, _ := range mockedLinks {
        _, err = appDbClient.HMSet("PORT_TABLE:"+k, ifStatusMock).Result()
        if err != nil {
            utils.PrintError("Error mocking admin and oper status for interface %s. Err: %v", k, err)
            return err
        } else {
            utils.PrintInfo("Successfully Mocked admin and oper status for interface %s", k)
        }
    }

    // Write first data points into redis.
    utils.PrintInfo("Counters Mock Initiated")
    var ifInErrors float64
    var ifInUnicastPackets float64
    var ifOutUnicastPackets float64
    var ifOutErrors float64
    ifInErrors = 100
    ifInUnicastPackets = 101
    ifOutUnicastPackets = 1100
    ifOutErrors = 1
    datapoint := map[string]interface{}{"SAI_PORT_STAT_IF_IN_ERRORS": ifInErrors, "SAI_PORT_STAT_IF_IN_UCAST_PKTS": ifInUnicastPackets, "SAI_PORT_STAT_IF_OUT_UCAST_PKTS": ifOutUnicastPackets, "SAI_PORT_STAT_IF_OUT_ERRORS": ifOutErrors}
    for ifName, oidMapping := range mockedLinks {
        _, err := countersDbClient.HMSet(counters_db+oidMapping, datapoint).Result()
        if err != nil {
            utils.PrintError("Error mocking redis data for interface %s. Err %v", ifName, err)
            return err
        } else {
            utils.PrintInfo("Successfuly mocked redis data for interface %s", ifName)
        }
    }
    time.Sleep(30 * time.Second)

    // Mock counters with sleep.
    for outlier := 0; outlier < len(outliers); outlier++ {
        if outliers[outlier] == 1 {
            ifInErrors = ifInErrors + 600
            ifInUnicastPackets = ifInUnicastPackets + 1005
            ifOutUnicastPackets = ifOutUnicastPackets + 1005
            ifOutErrors = ifOutErrors + 30
        } else {
            ifInErrors = ifInErrors + 200
            ifInUnicastPackets = ifInUnicastPackets + 300000015
            ifOutUnicastPackets = ifOutUnicastPackets + 1005
            ifOutErrors = ifOutErrors + 30
        }

        datapoint = map[string]interface{}{"SAI_PORT_STAT_IF_IN_ERRORS": ifInErrors, "SAI_PORT_STAT_IF_IN_UCAST_PKTS": ifInUnicastPackets, "SAI_PORT_STAT_IF_OUT_UCAST_PKTS": ifOutUnicastPackets, "SAI_PORT_STAT_IF_OUT_ERRORS": ifOutErrors}

        for ifName, oidMapping := range mockedLinks {
            _, err := countersDbClient.HMSet(counters_db+oidMapping, datapoint).Result()
            if err != nil {
                utils.PrintError("Error mocking redis data for interface %s. Err %v", ifName, err)
                return err
            } else {
                utils.PrintInfo("Successfuly mocked redis data for interface %s", ifName)
            }
        }
        time.Sleep(30 * time.Second)
    }
    utils.PrintInfo("Redis Mock Done")
    return nil
}
