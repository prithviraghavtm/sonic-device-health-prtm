package yang_utils

import (
    "testing"
    "encoding/json"
    "github.com/stretchr/testify/require"
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
	require.JSONEq(t, expectedActionsJson, string(resultActionsJson))

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
	require.JSONEq(t, expectedBindingsJson, string(resultBindingsJson))
}



