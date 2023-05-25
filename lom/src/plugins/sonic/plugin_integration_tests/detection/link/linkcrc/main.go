package main

import (
	"os/exec"
	"fmt"
	"github.com/go-redis/redis"
	"lom/src/plugins/sonic/plugin_integration_tests/utils"
        "lom/src/lib/lomcommon"
        "lom/src/lib/lomipc"
        "strconv"
        "time"
	"os"
	"io/ioutil"
	"lom/src/plugins/sonic/plugin/detection/link/linkcrc"
	"lom/src/plugins/plugins_common"
)

const (
	redis_address                    = "localhost:6379"
	redis_counters_db                = 2
	redis_app_db                     = 0
	fileName                         = "./COUNTERS_FOR_LINK_CRC_DATA_POINT"
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

        // Perform Actual integration test
	go InvokeLinkCrcDetectionPlugin()
	for {
	    select {
	    case <-time.After(3 * time.Minute):
		    utils.PrintError("Timeout. Aborting Integration test")
	       	    break
	    }
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

func InvokeLinkCrcDetectionPlugin() {
	go MockRedisData()
	linkCrcDetectionPlugin := linkcrc.LinkCRCDetectionPlugin{}
	actionCfg := lomcommon.ActionCfg_t{Name: action_name, Type: detection_type, Timeout: 0, HeartbeatInt: 10, Disable: false, Mimic: false, ActionKnobs: ""}
	linkCrcDetectionPlugin.Init(&actionCfg)
	actionRequest := lomipc.ActionRequestData{Action: action_name, InstanceId: "InstId", AnomalyInstanceId: "AnInstId", AnomalyKey: "", Timeout: 0}
	pluginHBChan := make(chan plugins_common.PluginHeartBeat, 10)
	go utils.ReceiveAndLogHeartBeat(pluginHBChan)
	time.Sleep(10 * time.Second)
	response := linkCrcDetectionPlugin.Request(pluginHBChan, &actionRequest)
	utils.PrintInfo("Integration testing Done.Anomaly detection result: %s", response.AnomalyKey)
}

func MockRedisData() error {
        datapoints := make([]map[string]interface{}, 5)

	for index := 0; index < 5; index++ {
		countersForLinkCRCBytes, err := ioutil.ReadFile(fileName + strconv.Itoa(index+1) + ".txt")
		if err != nil {
			utils.PrintError("Error reading file %d. Err %v", index+1, err)
			return err
		}
		datapoints[index] = utils.LoadConfigToMap(countersForLinkCRCBytes)
		fmt.Println(datapoints[index])
	}

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

	// Mock counters with sleep.
	utils.PrintInfo("Counters Mock Initiated")
	for datapointIndex := 0; datapointIndex < len(datapoints); datapointIndex++ {
		for ifName, oidMapping := range mockedLinks {
			_, err := countersDbClient.HMSet(counters_db+oidMapping, datapoints[datapointIndex]).Result()
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
