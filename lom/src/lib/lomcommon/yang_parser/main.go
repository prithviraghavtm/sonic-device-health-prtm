package main

import (
	"fmt"
	"lom/src/lib/lomcommon/yang_parser/yang_utils"
)
const (
	device_health_actions_configs      string = "device-health-actions-configs"
	device_health_actions_configs_path string = ".\\yang\\device-health-actions-configs.yang"

	device_health_bindings_configs      string = "device-health-bindings-configs"
	device_health_bindings_configs_path string = ".\\yang\\device-health-bindings-configs.yang"

	device_health_globals_configs      string = "device-health-global-configs"
	device_health_globals_configs_path string = ".\\yang\\device-health-global-configs.yang"

	device_health_procs_configs      string = "device-health-procs-configs"
	device_health_procs_configs_path string = ".\\yang\\device-health-procs-configs.yang"

	actions_conf_file_name  string = "actions.conf.json"
	bindings_conf_file_name string = "bindings.conf.json"
	globals_conf_file_name  string = "globals.conf.json"
	procs_conf_file_name    string = "procs.conf.json"
)


func main() {

        actionMapping, err := yang_utils.GetMappingForActionsYangConfig(device_health_actions_configs, device_health_actions_configs_path)
	if len(err) > 0 {
		fmt.Printf("Error getting mapping for Actions Yang config file")
		return
	}
	writeError := yang_utils.WriteJsonIntoFile(actionMapping, actions_conf_file_name)
	if writeError != nil {
		fmt.Printf("Writing actions conf failed.")
		return
	}

	bindingsMapping, err := yang_utils.GetMappingForBindingsYangConfig(device_health_bindings_configs, device_health_bindings_configs_path)
	if len(err) > 0 {
		fmt.Printf("Error getting mapping for Bindings Yang config file")
		return
	}
	writeError = yang_utils.WriteJsonIntoFile(bindingsMapping, bindings_conf_file_name)
	if writeError != nil {
		fmt.Printf("Writing bindings conf failed.")
		return
	}

	globalsMapping, err := yang_utils.GetMappingForGlobalsYangConfig(device_health_globals_configs, device_health_globals_configs_path)
	if len(err) > 0 {
		fmt.Printf("Error getting mapping for Globals Yang config file")
		return
	}
	writeError = yang_utils.WriteJsonIntoFile(globalsMapping, globals_conf_file_name)
	if writeError != nil {
		fmt.Printf("Writing globals conf failed.")
		return
	}

	procsMapping, err := yang_utils.GetMappingForProcsYangConfig(device_health_procs_configs, device_health_procs_configs_path)
	if len(err) > 0 {
		fmt.Printf("Error getting mapping for Procs Yang config file")
		return
	}
	writeError = yang_utils.WriteJsonIntoFile(procsMapping, procs_conf_file_name)
	if writeError != nil {
		fmt.Printf("Writing procs conf failed.")
		return
	}
}

