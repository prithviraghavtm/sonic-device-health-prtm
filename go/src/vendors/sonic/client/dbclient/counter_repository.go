// Defines APIs to get interface counters from redis db - CountersDB.
package dbclient

import (
	"strconv"
	. "go/src/lib/lomcommon"
)

const (
	counters_port_name_map_redis_key      string = "COUNTERS_PORT_NAME_MAP"
	counters_db_table_name                string = "COUNTERS:"
	sai_port_stat_if_in_errors_field      string = "SAI_PORT_STAT_IF_IN_ERRORS"
	sai_port_stat_if_in_ucast_pkts_field  string = "SAI_PORT_STAT_IF_IN_UCAST_PKTS"
	sai_port_stat_if_out_ucast_pkts_field string = "SAI_PORT_STAT_IF_OUT_UCAST_PKTS"
	if_in_errors_counter_key              string = "IfInErrors"
	in_unicast_packets_counter_key        string = "InUnicastPackets"
	out_unicast_packets_counter_key       string = "OutUnicastPackets"
	atoi_base                                int = 10
        uint_bit_size                            int = 64
)

type CounterRepository struct {
	RedisProvider RedisProviderInterface
}

/* Cache for storing interface to Oid mapping */
var interfaceToOidMapping map[string]string

/*
Returns interface counters for all interfaces on the Sonic device.
First it gets all oids for interfaces and then gets counters for each interface by performing redis hmGet calls.
*/
func (counterRepository *CounterRepository) GetInterfaceCounters() (map[string]map[string]uint64, error) {

	var interfaceCountersMap = make(map[string]map[string]uint64)
        var err error

	if interfaceToOidMapping == nil {
	   interfaceToOidMapping, err = counterRepository.RedisProvider.HGetAll(COUNTER_DB_ID, counters_port_name_map_redis_key)
	   if err != nil {
       		return nil, LogError("HGetAll Failed. err: (%v)", err)
           }
        }

	for interfaceName, interfaceOid := range interfaceToOidMapping {

		interfaceCountersKey := counters_db_table_name + interfaceOid
		fields := []string{sai_port_stat_if_in_errors_field, sai_port_stat_if_in_ucast_pkts_field, sai_port_stat_if_out_ucast_pkts_field}
		result, err := counterRepository.RedisProvider.HmGet(COUNTER_DB_ID, interfaceCountersKey, fields)

		if err != nil {
			return nil, LogError("HmGet for key (%s) failed. err:(%v)", interfaceCountersKey, err)
		}

		ifInErrors, err := strconv.ParseUint(result[0].(string), atoi_base, uint_bit_size)
		if err != nil {
			return nil, LogError("IfInErrors counter ParseUint conversion failed for key (%s). err: (%v)", interfaceCountersKey, err)
		}

		inUnicastPackets, err := strconv.ParseUint(result[1].(string), atoi_base, uint_bit_size)
		if err != nil {
			return nil, LogError("InUnicastPackets counter ParseUint conversion failed for key (%s). err: (%v)", interfaceCountersKey, err)
		}

		outUnicastPackets, err := strconv.ParseUint(result[2].(string), atoi_base, uint_bit_size)
		if err != nil {
			return nil, LogError("OutUnicastPackets counter ParseUint conversion failed for key (%s). err: (%v)", interfaceCountersKey, err)
		}

		var interfaceCounters = map[string]uint64{if_in_errors_counter_key: ifInErrors, in_unicast_packets_counter_key: inUnicastPackets, out_unicast_packets_counter_key: outUnicastPackets}
		interfaceCountersMap[interfaceName] = interfaceCounters
	}

	return interfaceCountersMap, nil
}
