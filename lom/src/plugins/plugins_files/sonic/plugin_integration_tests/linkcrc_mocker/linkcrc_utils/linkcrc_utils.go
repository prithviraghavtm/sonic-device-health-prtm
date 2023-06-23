package linkcrc_utils

import (
    "context"
    "github.com/go-redis/redis"
    "lom/src/plugins/sonic/plugin_integration_tests/utils"
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

func MockRedisWithLinkCrcCounters(pattern string, mockTimeInMinutes int, interfaces []string, ctx context.Context) {
    var countersDbClient = redis.NewClient(&redis.Options{
        Addr:     redis_address,
        Password: redis_password,
        DB:       redis_counters_db,
    })

    // Get all ifName to oid mappings.
    interfaceToOidMapping, err := countersDbClient.HGetAll(counters_port_name_map_redis_key).Result()
    if err != nil {
        utils.PrintError("Error fetching counters port name map %v", err)
        return
    }

    // Filter only those required as per the links passed through arguments
    mockedLinks := make(map[string]string)
    for _, v := range interfaces {
        mockedLinks[v] = interfaceToOidMapping[v]
    }

    // Set admin and oper status as up for mocked links.
    var appDbClient = redis.NewClient(&redis.Options{
        Addr:     redis_address,
        Password: redis_password,
        DB:       redis_app_db,
    })
    ifStatusMock := map[string]interface{}{admin_status: ifUp, oper_status: ifUp}
    for k, _ := range mockedLinks {
        _, err = appDbClient.HMSet("PORT_TABLE:"+k, ifStatusMock).Result()
        if err != nil {
            utils.PrintError("Error mocking admin and oper status for interface %s. Err: %v", k, err)
            return
        } else {
            utils.PrintInfo("Successfully Mocked admin and oper status for interface %s", k)
        }
    }

    var ifInErrors float64
    var ifInUnicastPackets float64
    var ifOutUnicastPackets float64
    var ifOutErrors float64
    ifInErrors = 100
    ifInUnicastPackets = 101
    ifOutUnicastPackets = 1100
    ifOutErrors = 1

    mockTimer := time.NewTimer(time.Duration(mockTimeInMinutes) * time.Minute)
    mockIntervalTicker := time.NewTicker(30 * time.Second)

    /* 0 indicates non outlier, 1 indicates outlier with non-zero counters */
    patternSlice := strings.Split(pattern, ",")
    patternLength := len(patternSlice)
    patternIndex := 0

loop:
    for {
        // Start mocking immediately
        datapoint := map[string]interface{}{"SAI_PORT_STAT_IF_IN_ERRORS": ifInErrors, "SAI_PORT_STAT_IF_IN_UCAST_PKTS": ifInUnicastPackets, "SAI_PORT_STAT_IF_OUT_UCAST_PKTS": ifOutUnicastPackets, "SAI_PORT_STAT_IF_OUT_ERRORS": ifOutErrors}
        for ifName, oidMapping := range mockedLinks {
            _, err := countersDbClient.HMSet(counters_db+oidMapping, datapoint).Result()
            if err != nil {
                utils.PrintError("Error mocking redis data for interface %s. Err %v", ifName, err)
                return
            } else {
                utils.PrintInfo("Successfuly mocked redis data for interface %s", ifName)
            }
        }

        if patternSlice[(patternIndex)%patternLength] == "1" {
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
        patternIndex++

        select {
        case <-mockTimer.C:
            break loop
        case <-mockIntervalTicker.C:
            continue
        case <-ctx.Done():
            break loop
        }
    }
    utils.PrintInfo("Done mocking redis")
}
