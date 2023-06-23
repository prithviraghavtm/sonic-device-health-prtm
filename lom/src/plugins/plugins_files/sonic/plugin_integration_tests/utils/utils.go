package utils

import (
    "encoding/json"
    "fmt"
    "lom/src/plugins/plugins_common"
)

const (
    integration_test_info_prefix = "[IntegrationTest][info]:"
    integration_test_err_prefix  = "[IntegrationTest][error]:"
)

func PrintInfo(str string, a ...any) {
    fmt.Printf(integration_test_info_prefix+str+"\n", a...)
}

func PrintError(str string, a ...any) {
    fmt.Printf(integration_test_err_prefix+str+"\n", a...)
}

func ReceiveAndLogHeartBeat(hbChannel chan plugins_common.PluginHeartBeat) {
    PrintInfo("Initiated HeartBeat Receiver")
    for index := 0; index < 100; index++ {
        <-hbChannel
        PrintInfo("Received heartbeat [%d]", index)
    }
}

func LoadConfigToMap(input []byte) map[string]interface{} {
    var mapping map[string]interface{}

    err := json.Unmarshal(input, &mapping)
    if err != nil {
        fmt.Println("Error un-marshalling bytes")
    }
    return mapping
}

func LoadConfigsToMap(input []byte) []map[string]interface{} {
    var mappings []map[string]interface{}

    err := json.Unmarshal(input, &mappings)
    if err != nil {
        fmt.Println("Error un-marshalling bytes to array of map")
    }
    return mappings
}
