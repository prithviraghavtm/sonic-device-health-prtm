package dbclient

import (
    "context"
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
    SAI_PORT_STAT_IF_OUT_ERRORS     = "SAI_PORT_STAT_IF_OUT_ERRORS"
    COUNTERS_PORT_NAME_MAP          = "COUNTERS_PORT_NAME_MAP"
    ADMIN_STATUS_FIELD              = "admin_status"
    OPER_STATUS_FIELD               = "oper_status"
    PORT_TABLE                      = "PORT_TABLE:"
)

func getInterfaceToODIMapping() map[string]string {
    newMap := map[string]string{
        ethernet1: oid1,
        ethernet2: oid2,
        ethernet3: oid3,
    }
    return newMap
}

func convertStringToInterfaceType(strs []string) []interface{} {
    counters := make([]interface{}, len(strs))
    for i, s := range strs {
        counters[i] = s
    }
    return counters
}

func validateOrderOfFields(fields []string) bool {
    return fields[0] == SAI_PORT_STAT_IF_IN_ERRORS && fields[1] == SAI_PORT_STAT_IF_IN_UCAST_PKTS && fields[2] == SAI_PORT_STAT_IF_OUT_UCAST_PKTS && fields[3] == SAI_PORT_STAT_IF_OUT_ERRORS
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

    strs := []string{"110", "111", "112", "113"}
    counters := convertStringToInterfaceType(strs)

    mockRedisProvider.On(HmGetMethod, 2, ethernet1_redis_key, mock.MatchedBy(validateOrderOfFields)).Return(counters, nil)

    strs1 := []string{"210", "211", "212", "213"}
    counters1 := convertStringToInterfaceType(strs1)
    mockRedisProvider.On(HmGetMethod, 2, ethernet2_redis_key, mock.MatchedBy(validateOrderOfFields)).Return(counters1, nil)

    strs2 := []string{"310", "311", "312", "313"}
    counters2 := convertStringToInterfaceType(strs2)
    mockRedisProvider.On(HmGetMethod, 2, ethernet3_redis_key, mock.MatchedBy(validateOrderOfFields)).Return(counters2, nil)

    // Act
    counterDBClient := CounterRepository{RedisProvider: mockRedisProvider}
    ctx, _ := context.WithCancel(context.Background())
    result, err := counterDBClient.GetCountersForAllInterfaces(ctx)

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
    assert.Equal(uint64(113), result[ethernet1]["IfOutErrors"], "Ehternet1's IfOutErrors counter is expected to be 113")
    assert.Equal(uint64(210), result[ethernet2]["IfInErrors"], "Ehternet2's IfInErrors counter is expected to be 210")
    assert.Equal(uint64(211), result[ethernet2]["InUnicastPackets"], "Ehternet2's InUnicastPackets counter is expected to be 211")
    assert.Equal(uint64(212), result[ethernet2]["OutUnicastPackets"], "Ehternet2's OutUnicastPackets counter is expected to be 212")
    assert.Equal(uint64(213), result[ethernet2]["IfOutErrors"], "Ehternet2's IfOutErrors counter is expected to be 213")
    assert.Equal(uint64(310), result[ethernet3]["IfInErrors"], "Ehternet3's IfInErrors counter is expected to be 310")
    assert.Equal(uint64(311), result[ethernet3]["InUnicastPackets"], "Ehternet3's InUnicastPackets counter is expected to be 311")
    assert.Equal(uint64(312), result[ethernet3]["OutUnicastPackets"], "Ehternet3's OutUnicastPackets counter is expected to be 312")
    assert.Equal(uint64(313), result[ethernet3]["IfOutErrors"], "Ehternet3's IfOutErrors counter is expected to be 313")
}

/* Test GetInterfaceCounters returns error when HGetAll method returns error */
func Test_GetInterfaceCounters_ReturnsErrorWhenHGetAllMethodFails(t *testing.T) {
    // Mock
    interfaceToOidMapping = nil
    mockRedisProvider := new(MockRedisProvider)
    (mockRedisProvider).On(HGetAllMethod, 2, COUNTERS_PORT_NAME_MAP).Return((map[string]string)(nil), errors.New("Error fetching data from redis."))

    // Act
    counterDBClient := CounterRepository{RedisProvider: mockRedisProvider}
    ctx, _ := context.WithCancel(context.Background())
    result, err := counterDBClient.GetCountersForAllInterfaces(ctx)

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

    strs := []string{"110", "111", "112", "113"}
    counters := convertStringToInterfaceType(strs)
    mockRedisProvider.On(HmGetMethod, 2, ethernet1_redis_key, mock.MatchedBy(validateOrderOfFields)).Maybe().Return(counters, nil)
    mockRedisProvider.On(HmGetMethod, 2, ethernet2_redis_key, mock.MatchedBy(validateOrderOfFields)).Return(([]interface{})(nil), errors.New("HmGet execution failed"))
    strs2 := []string{"310", "311", "312", "313"}
    counters2 := convertStringToInterfaceType(strs2)
    mockRedisProvider.On(HmGetMethod, 2, ethernet3_redis_key, mock.MatchedBy(validateOrderOfFields)).Maybe().Return(counters2, nil)

    // Act
    counterDBClient := CounterRepository{RedisProvider: mockRedisProvider}
    ctx, _ := context.WithCancel(context.Background())
    result, err := counterDBClient.GetCountersForAllInterfaces(ctx)

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

    strs := []string{"abc", "111", "112", "113"}
    counters := convertStringToInterfaceType(strs)
    mockRedisProvider.On(HmGetMethod, 2, ethernet1_redis_key, mock.MatchedBy(validateOrderOfFields)).Return(counters, nil)

    strs1 := []string{"210", "211", "212", "213"}
    counters1 := convertStringToInterfaceType(strs1)
    mockRedisProvider.On(HmGetMethod, 2, ethernet2_redis_key, mock.MatchedBy(validateOrderOfFields)).Maybe().Return(counters1, nil)

    strs2 := []string{"310", "311", "312", "313"}
    counters2 := convertStringToInterfaceType(strs2)
    mockRedisProvider.On(HmGetMethod, 2, ethernet3_redis_key, mock.MatchedBy(validateOrderOfFields)).Maybe().Return(counters2, nil)

    // Act
    counterDBClient := CounterRepository{RedisProvider: mockRedisProvider}
    ctx, _ := context.WithCancel(context.Background())
    result, err := counterDBClient.GetCountersForAllInterfaces(ctx)

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

    strs := []string{"110", "abc", "112", "113"}
    counters := convertStringToInterfaceType(strs)
    mockRedisProvider.On(HmGetMethod, 2, ethernet1_redis_key, mock.MatchedBy(validateOrderOfFields)).Return(counters, nil)

    strs1 := []string{"210", "211", "212", "213"}
    counters1 := convertStringToInterfaceType(strs1)
    mockRedisProvider.On(HmGetMethod, 2, ethernet2_redis_key, mock.MatchedBy(validateOrderOfFields)).Maybe().Return(counters1, nil)

    strs2 := []string{"310", "311", "312", "313"}
    counters2 := convertStringToInterfaceType(strs2)
    mockRedisProvider.On(HmGetMethod, 2, ethernet3_redis_key, mock.MatchedBy(validateOrderOfFields)).Maybe().Return(counters2, nil)

    // Act
    counterDBClient := CounterRepository{RedisProvider: mockRedisProvider}
    ctx, _ := context.WithCancel(context.Background())
    result, err := counterDBClient.GetCountersForAllInterfaces(ctx)

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

    strs := []string{"110", "111", "abc", "113"}
    counters := convertStringToInterfaceType(strs)
    mockRedisProvider.On(HmGetMethod, 2, ethernet1_redis_key, mock.MatchedBy(validateOrderOfFields)).Return(counters, nil)

    strs1 := []string{"210", "211", "212", "213"}
    counters1 := convertStringToInterfaceType(strs1)
    mockRedisProvider.On(HmGetMethod, 2, ethernet2_redis_key, mock.MatchedBy(validateOrderOfFields)).Maybe().Return(counters1, nil)

    strs2 := []string{"310", "311", "312", "313"}
    counters2 := convertStringToInterfaceType(strs2)
    mockRedisProvider.On(HmGetMethod, 2, ethernet3_redis_key, mock.MatchedBy(validateOrderOfFields)).Maybe().Return(counters2, nil)

    // Act
    counterDBClient := CounterRepository{RedisProvider: mockRedisProvider}
    ctx, _ := context.WithCancel(context.Background())
    result, err := counterDBClient.GetCountersForAllInterfaces(ctx)
    mockRedisProvider.AssertExpectations(t)

    // Assert
    if result != nil {
        t.Errorf("result is expected to be nil")
    }
    assert.NotEqual(t, nil, err, "err is exptected to be non nil")
}

func validateOrderOfStatusFields(fields []string) bool {
    return fields[0] == ADMIN_STATUS_FIELD && fields[1] == OPER_STATUS_FIELD
}

/* Validates isInterfaceActive returns true when both admin and oper status is up */
func Test_isInterfaceActive_ReturnsTrueForActiveInterface(t *testing.T) {
    // Mock
    mockRedisProvider := new(MockRedisProvider)
    ifStatuses := []string{"up", "up"}
    (mockRedisProvider).On(HmGetMethod, 0, PORT_TABLE+ethernet1, mock.MatchedBy(validateOrderOfStatusFields)).Return(convertStringToInterfaceType(ifStatuses), nil)
    // Act
    counterDBClient := CounterRepository{RedisProvider: mockRedisProvider}
    // Assert
    adminStatus, operStatus, err := counterDBClient.GetInterfaceStatus(ethernet1)
    assert.Equal(t, nil, err, "err is exptected to be nil")
    assert.True(t, adminStatus, "adminStatus is expected to be true")
    assert.True(t, operStatus, "operStatus is expected to be true")
}

/* Validates isInterfaceActive returns false when either admin and oper status is not up */
func Test_isInterfaceActive_ReturnsFalseForInActiveInterface(t *testing.T) {
    // Mock
    mockRedisProvider := new(MockRedisProvider)
    ifStatuses := []string{"down", "up"}
    (mockRedisProvider).On(HmGetMethod, 0, PORT_TABLE+ethernet1, mock.MatchedBy(validateOrderOfStatusFields)).Return(convertStringToInterfaceType(ifStatuses), nil)
    // Act
    counterDBClient := CounterRepository{RedisProvider: mockRedisProvider}
    // Assert
    adminStatus, operStatus, err := counterDBClient.GetInterfaceStatus(ethernet1)
    assert.Equal(t, nil, err, "err is exptected to be nil")
    assert.False(t, adminStatus, "adminStatus is exptected to be False")
    assert.True(t, operStatus, "operStatus is expected to be true")
}

/* Validates isInterfaceActive returns false when redis call fails */
func Test_isInterfaceActive_ReturnsFalseWhenRedisCallFails(t *testing.T) {
    // Mock
    mockRedisProvider := new(MockRedisProvider)
    (mockRedisProvider).On(HmGetMethod, 0, PORT_TABLE+ethernet1, mock.MatchedBy(validateOrderOfStatusFields)).Return(([]interface{})(nil), errors.New("HmGet execution failed"))
    // Act
    counterDBClient := CounterRepository{RedisProvider: mockRedisProvider}
    // Assert
    adminStatus, operStatus, err := counterDBClient.GetInterfaceStatus(ethernet1)
    assert.NotEqual(t, nil, err, "err is exptected to be nil")
    assert.False(t, adminStatus, "adminStatus is exptected to be False")
    assert.False(t, operStatus, "operStatus is exptected to be False")
}

/* Test GetInterfaceCounters returns error when context is cancelled. */
func Test_GetInterfaceCounters_ErrorsWhenContextCancelled(t *testing.T) {
    // Mock
    mockRedisProvider := new(MockRedisProvider)
    newMap := getInterfaceToODIMapping()
    (mockRedisProvider).On(HGetAllMethod, 2, COUNTERS_PORT_NAME_MAP).Maybe().Return(newMap, nil)

    // Act
    counterDBClient := CounterRepository{RedisProvider: mockRedisProvider}
    ctx, cancelFunc := context.WithCancel(context.Background())
    cancelFunc()
    result, err := counterDBClient.GetCountersForAllInterfaces(ctx)

    // Assert
    if result != nil {
        t.Errorf("result is expected to be nil")
    }
    assert.NotEqual(t, nil, err, "err is exptected to be non nil")
}
