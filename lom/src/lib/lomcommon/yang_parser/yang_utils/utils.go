package yang_utils

import (
    "encoding/json"
    "errors"
    "fmt"
    "github.com/openconfig/goyang/pkg/yang"
    "io/ioutil"
    "strconv"
)

const (
    actions_key_name      string = "Actions"
    bindings_key_name     string = "bindings"
    action_knobs_key_name string = "ActionKnobs"
    procs_key_name        string = "procs"
    int64_type            string = "int64"
    string_type           string = "string"
    boolean_type          string = "boolean"
    decimal64_type        string = "decimal64"
)

func ProcessLeafElements(leafElements map[string]*yang.Entry) (map[string]interface{}, error) {
    result := map[string]interface{}{}

    for leafName, leafElement := range leafElements {
        if leafElement.IsLeaf() && leafElement.Config == 1 {
            defaultValue := leafElement.Default
            configType := leafElement.Type.Name

            if configType == int64_type {
                val, err := strconv.ParseInt(defaultValue[0], 10, 64)
                if err != nil {
                    fmt.Printf("Error parsing int64. Err %v", err)
                    return nil, err
                }
                result[leafName] = val
            } else if configType == string_type {
                result[leafName] = defaultValue[0]
            } else if configType == boolean_type {
                val, err := strconv.ParseBool(defaultValue[0])
                if err != nil {
                    fmt.Printf("Error parsing boolean. Err %v", err)
                    return nil, err
                }
                result[leafName] = val
            } else if configType == decimal64_type {
                val, err := strconv.ParseFloat(defaultValue[0], 64)
                if err != nil {
                    fmt.Printf("Error parsing float. Err %v", err)
                    return nil, err
                }
                result[leafName] = val
            } else {
                fmt.Printf("Invalid leaf type")
                return nil, errors.New("invalid leaf type")
            }
        }
    }

    return result, nil
}

func GetMappingForGlobalsYangConfig(module string, yangFilePath string) (map[string]interface{}, []error) {
    entry, errs := yang.GetModule(module, yangFilePath)

    if len(errs) > 0 {
        fmt.Printf("invalid device-health-global-configs yang file. Err %v", errs)
        return nil, errs
    }

    configMap, err := ProcessLeafElements(entry.Dir)

    if err != nil {
        fmt.Printf("Error processing leaf elements for Globals Yang Config. Err %v", err)
        return nil, []error{err}
    }
    return configMap, nil
}

func GetMappingForProcsYangConfig(module string, yangFilePath string) (map[string]interface{}, []error) {
    entry, errs := yang.GetModule(module, yangFilePath)

    if len(errs) > 0 {
        fmt.Printf("invalid device-health-global-configs yang file. Err %v", errs)
        return nil, errs
    }

    finalMap := map[string]interface{}{}
    for containerName, containerElement := range entry.Dir {

        if !containerElement.IsContainer() {
            errorStr := fmt.Sprintf("procs - intiial level is expected to be a container. ContainerName %s", containerName)
            fmt.Println(errorStr)
            return nil, []error{errors.New(errorStr)}
        }

        subMap := map[string]interface{}{}
        for subContName, subContElement := range containerElement.Dir {
            if !subContElement.IsContainer() {
                errorStr := fmt.Sprintf("procs - second layer is expected to ba a container. subContName %s", subContName)
                fmt.Println(errorStr)
                return nil, []error{errors.New(errorStr)}
            }

            leafMap, err := ProcessLeafElements(subContElement.Dir)
            if err != nil {
                fmt.Printf("Error processing leaf elements for subContName %s. Err %v", subContName, err)
                return nil, []error{err}
            }
            subMap[subContName] = leafMap
        }

        finalMap[containerName] = subMap
    }

    resultMap := map[string]interface{}{procs_key_name: finalMap}
    return resultMap, nil
}

func GetMappingForActionsYangConfig(module string, yangFilePath string) (map[string]interface{}, []error) {
    entry, errs := yang.GetModule(module, yangFilePath)

    if len(errs) > 0 {
        fmt.Printf("Invalid device-health-global-configs yang file. Err %v", errs)
        return nil, errs
    }

    finalMap := map[string]interface{}{}
    for containerName, containerElement := range entry.Dir {

        leafMap, err := ProcessLeafElements(containerElement.Dir)
        if err != nil {
            fmt.Printf("Actions - Error processing leaf elements for containerName %s. Err %v", containerName, err)
            return nil, []error{err}
        }

        for subContName, subContElement := range containerElement.Dir {
            if subContElement.IsContainer() {
                if subContName != action_knobs_key_name {
                    errorStr := fmt.Sprintf("Actions - Invalid yang schema with invalid subContName %s", subContName)
                    fmt.Println(errorStr)
                    return nil, []error{errors.New(errorStr)}
                }
                actionKnobsMap, err := ProcessLeafElements(subContElement.Dir)
                if err != nil {
                    fmt.Printf("Actions - Error processing leaf elements for subContName %s. Err %v", subContName, err)
                    return nil, []error{err}
                }

                leafMap[subContName] = actionKnobsMap
                break
            }
        }

        finalMap[containerName] = leafMap
    }
    return finalMap, nil
}

func GetMappingForBindingsYangConfig(module string, yangFilePath string) (map[string]interface{}, []error) {
    entry, errs := yang.GetModule(module, yangFilePath)

    if len(errs) > 0 {
        fmt.Printf("Invalid device-health-global-configs yang file. Err %v", errs)
        return nil, errs
    }

    finalMap := []map[string]interface{}{}
    for containerName, containerElement := range entry.Dir {

        leafMap, err := ProcessLeafElements(containerElement.Dir)
        if err != nil {
            fmt.Printf("Global - Error processing leaf elements for containerName %s. Err %v", containerName, err)
            return nil, []error{err}
        }

        for subContName, subContElement := range containerElement.Dir {
            if subContElement.IsContainer() {
                if subContName != actions_key_name {
                    errorStr := fmt.Sprintf("Bindings - Invalid yang schema with invalid subContName %s", subContName)
                    fmt.Println(errorStr)
                    return nil, []error{errors.New(errorStr)}
                }

                listOfMaps := []map[string]interface{}{}
                for contName, contElement := range subContElement.Dir {
                    actionKnobsMap, err := ProcessLeafElements(contElement.Dir)
                    if err != nil {
                        fmt.Printf("Bindings - Error processing leaf elements for contName %s. Err %v", contName, err)
                        return nil, []error{err}
                    }

                    listOfMaps = append(listOfMaps, actionKnobsMap)
                }

                leafMap[subContName] = listOfMaps
                break
            }
        }

        finalMap = append(finalMap, leafMap)
    }

    resultMap := map[string]interface{}{}
    resultMap[bindings_key_name] = finalMap
    return resultMap, nil
}

func WriteJsonIntoFile(mapping map[string]interface{}, folder string, fileName string) error {
    jsonConfig, err := json.MarshalIndent(mapping, "  ", "  ")

    if err != nil {
        fmt.Printf("Error marshalling mapping to json string. Err %v", err)
        return err
    }

    fmt.Println(string(jsonConfig))
    err = ioutil.WriteFile(folder+"/"+fileName, jsonConfig, 0644)

    if err != nil {
        fmt.Printf("Error writing json into file. Err %v", err)
        return err
    }

    return nil
}
