// Defines APIs to get interface counters from redis db - CountersDB.
package dbclient

import (
    "strconv"
    "fmt"
    "errors"
)

const (
    counters_port_name_map_redis_key      string = "COUNTERS_PORT_NAME_MAP"
    counters_db_table_name                string = "COUNTERS:"
    sai_port_stat_if_in_errors_field      string = "SAI_PORT_STAT_IF_IN_ERRORS"
    sai_port_stat_if_out_errors_field     string = "SAI_PORT_STAT_IF_OUT_ERRORS"
    sai_port_stat_if_in_ucast_pkts_field  string = "SAI_PORT_STAT_IF_IN_UCAST_PKTS"
    sai_port_stat_if_out_ucast_pkts_field string = "SAI_PORT_STAT_IF_OUT_UCAST_PKTS"
    atoi_base                                int = 10
    uint_bit_size                            int = 64
    port_table_redis_key                  string = "PORT_TABLE:"
    admin_status_field                    string = "admin_status"
    oper_status_field                     string = "oper_status"
    interface_status_up                   string = "up"
    IF_IN_ERRORS_COUNTER_KEY              string = "IfInErrors"
    IF_OUT_ERRORS_COUNTER_KEY             string = "IfOutErrors"
    IN_UNICAST_PACKETS_COUNTER_KEY        string = "InUnicastPackets"
    OUT_UNICAST_PACKETS_COUNTER_KEY       string = "OutUnicastPackets"
)

type CounterRepository struct {
    RedisProvider RedisProviderInterface
}

type InterfaceCountersMap map[string]map[string]uint64

/* Cache for storing interface to Oid mapping */
var interfaceToOidMapping map[string]string

type CounterRepositoryInterface interface {
    GetCountersForActiveInterfaces() (InterfaceCountersMap, error)
    IsInterfaceActive(interfaceName string) (bool, error)
}

/*
Returns interface counters for all interfaces on the Sonic device.
First it gets all oids for interfaces and then gets counters for each interface by performing redis hmGet calls.
*/
func (counterRepository *CounterRepository) GetCountersForActiveInterfaces() (InterfaceCountersMap, error) {

    var interfaceCountersMap = make(InterfaceCountersMap)
        var err error

    if interfaceToOidMapping == nil {
       interfaceToOidMapping, err = counterRepository.RedisProvider.HGetAll(COUNTER_DB_ID, counters_port_name_map_redis_key)
       if err != nil {
               return nil, errors.New(fmt.Sprintf("HGetAll Failed. err: (%v)", err))
           }
        }

    for interfaceName, interfaceOid := range interfaceToOidMapping {
        isInterfaceActive, err := counterRepository.IsInterfaceActive(interfaceName)
        if err != nil {
            return nil, err
        }

        if isInterfaceActive {
            interfaceCountersKey := counters_db_table_name + interfaceOid
            fields := []string{sai_port_stat_if_in_errors_field, sai_port_stat_if_in_ucast_pkts_field, sai_port_stat_if_out_ucast_pkts_field, sai_port_stat_if_out_errors_field}
            result, err := counterRepository.RedisProvider.HmGet(COUNTER_DB_ID, interfaceCountersKey, fields)

            if err != nil {
                return nil, errors.New(fmt.Sprintf("HmGet for key (%s) failed. err:(%v)", interfaceCountersKey, err))
            }

            ifInErrors, err := strconv.ParseUint(result[0].(string), atoi_base, uint_bit_size)
            if err != nil {
                return nil, errors.New(fmt.Sprintf("IfInErrors counter ParseUint conversion failed for key (%s). err: (%v)", interfaceCountersKey, err))
            }

            inUnicastPackets, err := strconv.ParseUint(result[1].(string), atoi_base, uint_bit_size)
            if err != nil {
                return nil, errors.New(fmt.Sprintf("InUnicastPackets counter ParseUint conversion failed for key (%s). err: (%v)", interfaceCountersKey, err))
            }

            outUnicastPackets, err := strconv.ParseUint(result[2].(string), atoi_base, uint_bit_size)
            if err != nil {
                return nil, errors.New(fmt.Sprintf("OutUnicastPackets counter ParseUint conversion failed for key (%s). err: (%v)", interfaceCountersKey, err))
            }

	    ifOutErrors, err := strconv.ParseUint(result[3].(string), atoi_base, uint_bit_size)
	    if err != nil {
		return nil, errors.New(fmt.Sprintf("IfOutErrors counter ParseUint conversion failed for key (%s). err: (%v)", interfaceCountersKey, err))
	    }

            var interfaceCounters = map[string]uint64{IF_IN_ERRORS_COUNTER_KEY: ifInErrors, IN_UNICAST_PACKETS_COUNTER_KEY: inUnicastPackets, OUT_UNICAST_PACKETS_COUNTER_KEY: outUnicastPackets, IF_OUT_ERRORS_COUNTER_KEY: ifOutErrors}
            interfaceCountersMap[interfaceName] = interfaceCounters
        }
    }

    return interfaceCountersMap, nil
}

/* Returns true if an interface's oper and admin status is up, else false. */
func (counterRepository *CounterRepository) IsInterfaceActive(interfaName string) (bool, error) {
    interfaceStatusKey := port_table_redis_key + interfaName
    fields := []string{admin_status_field, oper_status_field}
    result, err := counterRepository.RedisProvider.HmGet(APPL_DB_ID, interfaceStatusKey, fields)
    if err != nil {
        return false, errors.New(fmt.Sprintf("isInterfaceActive.HmGet Failed for key (%s). err: (%v)", interfaceStatusKey, err))
    }
    if result[0].(string) == interface_status_up && result[1].(string) == interface_status_up {
        return true, nil
    }
    return false, nil
}


