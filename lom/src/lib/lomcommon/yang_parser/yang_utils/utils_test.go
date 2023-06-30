package yang_utils

import (
    "encoding/json"
    "github.com/stretchr/testify/require"
    "testing"
    "fmt"
    "github.com/stretchr/testify/assert"
    "github.com/openconfig/goyang/pkg/yang"
)

func Test_GetMappingForAllYangConfig_GeneratesCorrectMapping(t *testing.T) {

    expectedActionsJson := `{
        "link_crc": {
          "ActionKnobs": {
            "DetectionFreqInSecs": 30,
            "IfInErrorsDiffMinValue": 0,
            "InUnicastPacketsMinValue": 100,
            "LookBackPeriodInSecs": 125,
            "MinCrcError": 0.000001,
            "MinOutliersForDetection": 2,
            "OutUnicastPacketsMinValue": 100,
            "OutlierRollingWindowSize": 5
          },
          "Disable": false,
          "HeartbeatInt": 30,
          "Mimic": false,
          "Name": "link_crc",
          "Timeout": 0,
          "Type": "Detection"
        }
      }`

    resultActionsMapping, _ := GetMappingForActionsYangConfig("device-health-actions-configs", "../yang_prod_configs/device-health-actions-configs.yang")
    resultActionsJson, _ := json.Marshal(resultActionsMapping)
    require.JSONEq(t, expectedActionsJson, string(resultActionsJson), "Generated Actions json is not as expected")

    expectedBindingsJson := `{
        "bindings": [
          {
            "Actions": [
              {
                "name": "link_crc"
              }
            ],
            "Priority": 0,
            "SequenceName": "link_crc_bind-0",
            "Timeout": 2
          }
        ]
      }`

    resultBindingsMapping, _ := GetMappingForBindingsYangConfig("device-health-bindings-configs", "../yang_prod_configs/device-health-bindings-configs.yang")
    resultBindingsJson, _ := json.Marshal(resultBindingsMapping)
    require.JSONEq(t, expectedBindingsJson, string(resultBindingsJson), "Generated Bindings json is not as expected")

    expectedGlobalsJson := `{
        "ENGINE_HB_INTERVAL_SECS": 10,
        "INITIAL_DETECTION_REPORTING_FREQ_IN_MINS": 5,
        "INITIAL_DETECTION_REPORTING_MAX_COUNT": 12,
        "MAX_PLUGIN_RESPONSES": 100,
        "MAX_PLUGIN_RESPONSES_WINDOW_TIMEOUT_IN_SECS": 60,
        "MAX_SEQ_TIMEOUT_SECS": 120,
        "MIN_PERIODIC_LOG_PERIOD_SECS": 1,
        "PLUGIN_MIN_ERR_CNT_TO_SKIP_HEARTBEAT": 3,
        "SUBSEQUENT_DETECTION_REPORTING_FREQ_IN_MINS": 60
      }`

    resultGlobalsMapping, _ := GetMappingForGlobalsYangConfig("device-health-global-configs", "../yang_prod_configs/device-health-global-configs.yang")
    resultGlobalsJson, _ := json.Marshal(resultGlobalsMapping)
    require.JSONEq(t, expectedGlobalsJson, string(resultGlobalsJson), "Generated Globals json is not as expected")

    expectedProcsJson := `{
        "procs": {
          "proc_0": {
            "link_crc": {
              "name": "link_crc",
              "path": "",
              "version": "1.0.0.0"
            }
          }
        }
      }`

    resultProcsMapping, _ := GetMappingForProcsYangConfig("device-health-procs-configs", "../yang_prod_configs/device-health-procs-configs.yang")
    resultProcsJson, _ := json.Marshal(resultProcsMapping)
    require.JSONEq(t, expectedProcsJson, string(resultProcsJson), "Generated Procs json is not as expected")
}

func Test_ProcessLeafElements_ReturnsErrorForInvalidValueTypes(t *testing.T) {

	modules := []string{"globals-invalid-int64-type-value", "globals-invalid-boolean-type-value", "globals-invalid-float-type-value"}
	paths := []string{"./yang_test_files/globals-invalid-int64-type-value.yang", "./yang_test_files/globals-invalid-boolean-type-value.yang", "./yang_test_files/globals-invalid-float-type-value.yang"}

	for index := 0; index < len(modules); index++ {
		globalsMapping, err := yang.GetModule(modules[index], paths[index])
		str := fmt.Sprintf("No error expected %d", index)
		assert.Equal(t, 0, len(err), str)
		leafMap, er := ProcessLeafElements(globalsMapping.Dir)
		assert.Equal(t, map[string]interface{}(nil), leafMap, fmt.Sprintf("Leafmap is expected to be nil for index %d", index))
		assert.NotEqual(t, nil, er, fmt.Sprintf("Error expected for index %d", index))
	}
}
