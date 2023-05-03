package dbclient

import (
        "errors"
        "testing"

        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/mock"
)

const (
        ethernet1                       = "ethernet1"
        ethernet2                       = "ethernet2"
        ethernet3                       = "ethernet3"
        oid1                            = "oid-1"
        oid2                            = "oid-2"
        oid3                            = "oid-3"
        ethernet1_redis_key             = "COUNTERS:oid-1"
        ethernet2_redis_key             = "COUNTERS:oid-2"
        ethernet3_redis_key             = "COUNTERS:oid-3"
        HGetAllMethod                   = "HGetAll"
        HmGetMethod                     = "HmGet"
        SAI_PORT_STAT_IF_IN_ERRORS      = "SAI_PORT_STAT_IF_IN_ERRORS"
        SAI_PORT_STAT_IF_IN_UCAST_PKTS  = "SAI_PORT_STAT_IF_IN_UCAST_PKTS"
        SAI_PORT_STAT_IF_OUT_UCAST_PKTS = "SAI_PORT_STAT_IF_OUT_UCAST_PKTS"
        COUNTERS_PORT_NAME_MAP          = "COUNTERS_PORT_NAME_MAP"
)

func getInterfaceToODIMapping() map[string]string {
        newMap := map[string]string{
                ethernet1: oid1,
                ethernet2: oid2,
                ethernet3: oid3,
        }
        return newMap
}

func getCountersForInterfaces(strs []string) []interface{} {
        counters := make([]interface{}, len(strs))
        for i, s := range strs {
                counters[i] = s
        }
        return counters
}

func validateOrderOfFields(fields []string) bool {
        return fields[0] == SAI_PORT_STAT_IF_IN_ERRORS && fields[1] == SAI_PORT_STAT_IF_IN_UCAST_PKTS && fields[2] == SAI_PORT_STAT_IF_OUT_UCAST_PKTS
}

type MockRedisProvider struct {
        mock.Mock
}

func (mockRedisProvider *MockRedisProvider) HmGet(database int, key string, fields []string) ([]interface{}, error) {
        args := mockRedisProvider.Called(database, key, fields)
        return args.Get(0).([]interface{}), args.Error(1)
}

func (mockRedisProvider *MockRedisProvider) HGetAll(database int, key string) (map[string]string, error) {
        args := mockRedisProvider.Called(database, key)
        return args.Get(0).(map[string]string), args.Error(1)
}

/* Test GetInterfaceCounters Method returns all counters successfuly */
func Test_GetInterfaceCounters_ReturnsAllCountersSuccessfuly(t *testing.T) {
        // Mock
        mockRedisProvider := new(MockRedisProvider)
        newMap := getInterfaceToODIMapping()
        (mockRedisProvider).On(HGetAllMethod, 2, COUNTERS_PORT_NAME_MAP).Maybe().Return(newMap, nil)

        strs := []string{"110", "111", "112"}
        counters := getCountersForInterfaces(strs)

        mockRedisProvider.On(HmGetMethod, 2, ethernet1_redis_key, mock.MatchedBy(validateOrderOfFields)).Return(counters, nil)

        strs1 := []string{"210", "211", "212"}
        counters1 := getCountersForInterfaces(strs1)
        mockRedisProvider.On(HmGetMethod, 2, ethernet2_redis_key, mock.MatchedBy(validateOrderOfFields)).Return(counters1, nil)

        strs2 := []string{"310", "311", "312"}
        counters2 := getCountersForInterfaces(strs2)
        mockRedisProvider.On(HmGetMethod, 2, ethernet3_redis_key, mock.MatchedBy(validateOrderOfFields)).Return(counters2, nil)

        // Act
        counterDBClient := CounterRepository{RedisProvider: mockRedisProvider}
        result, err := counterDBClient.GetInterfaceCounters()

        // Assert
        mockRedisProvider.AssertNumberOfCalls(t, HmGetMethod, 3)
        mockRedisProvider.AssertExpectations(t)
        assert := assert.New(t)
        assert.NotEqual(nil, result, "Result is expected to be non nil")
        assert.Equal(nil, err, "GetInterfaceCounters: Error is expected to be nil")
        assert.Equal(3, len(result), "GetInterfaceCounters: length of resulting map is expected to be 3")
        assert.Equal(uint64(110), result[ethernet1]["IfInErrors"], "Ehternet1's IfInErrors counter is expected to be 110")
        assert.Equal(uint64(111), result[ethernet1]["InUnicastPackets"], "Ehternet1's InUnicastPackets counter is expected to be 111")
        assert.Equal(uint64(112), result[ethernet1]["OutUnicastPackets"], "Ehternet1's OutUnicastPackets counter is expected to be 112")
        assert.Equal(uint64(210), result[ethernet2]["IfInErrors"], "Ehternet2's IfInErrors counter is expected to be 210")
        assert.Equal(uint64(211), result[ethernet2]["InUnicastPackets"], "Ehternet2's InUnicastPackets counter is expected to be 211")
        assert.Equal(uint64(212), result[ethernet2]["OutUnicastPackets"], "Ehternet2's OutUnicastPackets counter is expected to be 212")
        assert.Equal(uint64(310), result[ethernet3]["IfInErrors"], "Ehternet3's IfInErrors counter is expected to be 310")
        assert.Equal(uint64(311), result[ethernet3]["InUnicastPackets"], "Ehternet3's InUnicastPackets counter is expected to be 311")
        assert.Equal(uint64(312), result[ethernet3]["OutUnicastPackets"], "Ehternet3's OutUnicastPackets counter is expected to be 312")
}

/* Test GetInterfaceCounters returns error when HGetAll method returns error */
func Test_GetInterfaceCounters_ReturnsErrorWhenHGetAllMethodFails(t *testing.T) {
        // Mock
        interfaceToOidMapping = nil
        mockRedisProvider := new(MockRedisProvider)
        (mockRedisProvider).On(HGetAllMethod, 2, COUNTERS_PORT_NAME_MAP).Return((map[string]string)(nil), errors.New("Error fetching data from redis."))

        // Act
        counterDBClient := CounterRepository{RedisProvider: mockRedisProvider}
        result, err := counterDBClient.GetInterfaceCounters()

        // Assert
        if result != nil {
           t.Errorf("result is expected to be nil")
        }
        assert.NotEqual(t, nil, err, "err is exptected to be non nil")
}

/* Test GetInterfaceCounters returns error when hmget call errors */
func Test_GetInterfaceCounters_ReturnsErrorWhenHmGetFails(t *testing.T) {
        // Mock
        mockRedisProvider := new(MockRedisProvider)
        newMap := getInterfaceToODIMapping()
        (mockRedisProvider).On(HGetAllMethod, 2, COUNTERS_PORT_NAME_MAP).Maybe().Return(newMap, nil)

        strs := []string{"110", "111", "112"}
        counters := getCountersForInterfaces(strs)
        mockRedisProvider.On(HmGetMethod, 2, ethernet1_redis_key, mock.MatchedBy(validateOrderOfFields)).Maybe().Return(counters, nil)
        mockRedisProvider.On(HmGetMethod, 2, ethernet2_redis_key, mock.MatchedBy(validateOrderOfFields)).Return(([]interface{})(nil), errors.New("HmGet execution failed"))
        strs2 := []string{"310", "311", "312"}
        counters2 := getCountersForInterfaces(strs2)
        mockRedisProvider.On(HmGetMethod, 2, ethernet3_redis_key, mock.MatchedBy(validateOrderOfFields)).Maybe().Return(counters2, nil)

        // Act
        counterDBClient := CounterRepository{RedisProvider: mockRedisProvider}
        result, err := counterDBClient.GetInterfaceCounters()

        // Assert
        mockRedisProvider.AssertExpectations(t)
        if result != nil {
                t.Errorf("result is expected to be nil")
        }
        assert.NotEqual(t, nil, err, "err is exptected to be non nil")
}

/* Test GetInterfaceCounters returns error for an invalid IfInErrors counter from redis */
func Test_GetInterfaceCounters_ReturnsErrorWhenIfInErrorsCastingFails(t *testing.T) {
        // Mock
        mockRedisProvider := new(MockRedisProvider)
        newMap := getInterfaceToODIMapping()
        (mockRedisProvider).On(HGetAllMethod, 2, COUNTERS_PORT_NAME_MAP).Maybe().Return(newMap, nil)

        strs := []string{"abc", "111", "112"}
        counters := getCountersForInterfaces(strs)
        mockRedisProvider.On(HmGetMethod, 2, ethernet1_redis_key, mock.MatchedBy(validateOrderOfFields)).Return(counters, nil)

        strs1 := []string{"210", "211", "212"}
        counters1 := getCountersForInterfaces(strs1)
        mockRedisProvider.On(HmGetMethod, 2, ethernet2_redis_key, mock.MatchedBy(validateOrderOfFields)).Maybe().Return(counters1, nil)

        strs2 := []string{"310", "311", "312"}
        counters2 := getCountersForInterfaces(strs2)
        mockRedisProvider.On(HmGetMethod, 2, ethernet3_redis_key, mock.MatchedBy(validateOrderOfFields)).Maybe().Return(counters2, nil)

        // Act
        counterDBClient := CounterRepository{RedisProvider: mockRedisProvider}
        result, err := counterDBClient.GetInterfaceCounters()

        // Assert
        mockRedisProvider.AssertExpectations(t)
        if result != nil {
                t.Errorf("result is expected to be nil")
        }
        assert.NotEqual(t, nil, err, "err is exptected to be non nil")
}

/* Test GetInterfaceCounters returns error for an invalid InUnicastPacket counter */
func Test_GetInterfaceCounters_ReturnsErrorWhenInUnicastPacketsCastingFails(t *testing.T) {
        // Mock
        mockRedisProvider := new(MockRedisProvider)
        newMap := getInterfaceToODIMapping()
        (mockRedisProvider).On(HGetAllMethod, 2, COUNTERS_PORT_NAME_MAP).Maybe().Return(newMap, nil)

        strs := []string{"110", "abc", "112"}
        counters := getCountersForInterfaces(strs)
        mockRedisProvider.On(HmGetMethod, 2, ethernet1_redis_key, mock.MatchedBy(validateOrderOfFields)).Return(counters, nil)

        strs1 := []string{"210", "211", "212"}
        counters1 := getCountersForInterfaces(strs1)
        mockRedisProvider.On(HmGetMethod, 2, ethernet2_redis_key, mock.MatchedBy(validateOrderOfFields)).Maybe().Return(counters1, nil)

        strs2 := []string{"310", "311", "312"}
        counters2 := getCountersForInterfaces(strs2)
        mockRedisProvider.On(HmGetMethod, 2, ethernet3_redis_key, mock.MatchedBy(validateOrderOfFields)).Maybe().Return(counters2, nil)

        // Act
        counterDBClient := CounterRepository{RedisProvider: mockRedisProvider}
        result, err := counterDBClient.GetInterfaceCounters()

        // Assert
        mockRedisProvider.AssertExpectations(t)
        if result != nil {
                t.Errorf("result is expected to be nil")
        }
        assert.NotEqual(t, nil, err, "err is exptected to be non nil")
}

/* Test GetInterfaceCounters returns error for an invalid OutUnicastPackets counter from redis */
func Test_GetInterfaceCounters_ReturnsErrorWhenOutUnicastPacketsCastingFails(t *testing.T) {
        // Mock
        mockRedisProvider := new(MockRedisProvider)
        newMap := getInterfaceToODIMapping()
        (mockRedisProvider).On(HGetAllMethod, 2, COUNTERS_PORT_NAME_MAP).Maybe().Return(newMap, nil)

        strs := []string{"110", "111", "abc"}
        counters := getCountersForInterfaces(strs)
        mockRedisProvider.On(HmGetMethod, 2, ethernet1_redis_key, mock.MatchedBy(validateOrderOfFields)).Return(counters, nil)

        strs1 := []string{"210", "211", "212"}
        counters1 := getCountersForInterfaces(strs1)
        mockRedisProvider.On(HmGetMethod, 2, ethernet2_redis_key, mock.MatchedBy(validateOrderOfFields)).Maybe().Return(counters1, nil)

        strs2 := []string{"310", "311", "312"}
        counters2 := getCountersForInterfaces(strs2)
        mockRedisProvider.On(HmGetMethod, 2, ethernet3_redis_key, mock.MatchedBy(validateOrderOfFields)).Maybe().Return(counters2, nil)

        // Act
        counterDBClient := CounterRepository{RedisProvider: mockRedisProvider}
        result, err := counterDBClient.GetInterfaceCounters()
        mockRedisProvider.AssertExpectations(t)

        // Assert
        if result != nil {
                t.Errorf("result is expected to be nil")
        }
        assert.NotEqual(t, nil, err, "err is exptected to be non nil")
}

