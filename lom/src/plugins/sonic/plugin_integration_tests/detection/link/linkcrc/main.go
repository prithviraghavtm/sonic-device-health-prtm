package main

import (
        "fmt"
        "github.com/go-redis/redis"
        "io/ioutil"
	"os"
        "encoding/json"
	"lom/src/plugins/plugins_common"
	"time"
        "lom/src/lib/lomcommon"
        "lom/src/lib/lomipc"
	"lom/src/plugins/sonic/plugin/detection/link/linkcrc"
)

func main() {
go MockRedisData()

	linkCrcDetectionPlugin := linkcrc.LinkCRCDetectionPlugin{}
	actionCfg := lomcommon.ActionCfg_t{Name: "link_crc_integration", Type: "detection", Timeout: 0, HeartbeatInt: 10, Disable: false, Mimic: false, ActionKnobs: ""}
	linkCrcDetectionPlugin.Init(&actionCfg)
	actionRequest := lomipc.ActionRequestData{Action: "link_crc_detection", InstanceId: "InstId", AnomalyInstanceId: "AnInstId", AnomalyKey: "", Timeout: 0}
	pluginHBChan := make(chan plugins_common.PluginHeartBeat, 10)
	go LogHeartBeat(pluginHBChan)
	fmt.Println("Started HeartBeat Receiver")
	time.Sleep(10 * time.Second)
	response := linkCrcDetectionPlugin.Request(pluginHBChan, &actionRequest)
	fmt.Println("Integration testing.Anomaly detection result: " + response.AnomalyKey)
	time.Sleep(30 * time.Minute)

}


func LogHeartBeat(hbChannel chan plugins_common.PluginHeartBeat) {
	for index := 0; index < 100; index++ {
		<-hbChannel
		fmt.Printf("Received heartbeat [%d]", index)
	}
}

func MockRedisData() error {
	datapoints := make([]map[string]interface{}, 5)
	fileNames := [5]string{"./COUNTERS_FOR_LINK_CRC_DATA_POINT1.txt", "./COUNTERS_FOR_LINK_CRC_DATA_POINT2.txt", "./COUNTERS_FOR_LINK_CRC_DATA_POINT3.txt", "./COUNTERS_FOR_LINK_CRC_DATA_POINT4.txt", "./COUNTERS_FOR_LINK_CRC_DATA_POINT5.txt"}

	for index := 0; index < 5; index++ {
		countersForLinkCRCBytes, err := ioutil.ReadFile(fileNames[index])
		if err != nil {
			fmt.Printf("Error reading file %s", fileNames[index])
			return err
		}
		datapoints[index] = loadConfig(countersForLinkCRCBytes)
		fmt.Println(datapoints[index])
	}

	var client = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       2,
	})

	for datapointIndex := 0; datapointIndex < len(datapoints); datapointIndex++ {
		for interfaceIndex := 0; interfaceIndex < len(os.Args) - 1; interfaceIndex++ {
			_, err := client.HMSet(os.Args[interfaceIndex + 1], datapoints[datapointIndex]).Result()
			if err != nil {
				fmt.Printf("Error mocking redis data for index %d and interface %d", datapointIndex, interfaceIndex)
				return err
			} else {
				fmt.Printf("Successfuly mocked redis data: %d and interface %d", datapointIndex, interfaceIndex)
			}
		}
		time.Sleep(30 * time.Second)
	}
        fmt.Println("Done mocking redis")
	return nil
}

func loadConfig(input []byte) map[string]interface{} {
	var mapping map[string]interface{}

	err := json.Unmarshal(input, &mapping)
	if err != nil {
		fmt.Println("Error un-marshalling bytes")
	}
	return mapping
}
