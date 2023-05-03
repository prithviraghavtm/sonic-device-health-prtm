package dbclient

import (
    "errors"
    "testing"

    "github.com/alicebob/miniredis"
    "github.com/go-redis/redis"
    "github.com/stretchr/testify/assert"
)

func mockHmGetFunction(redisClient *redis.Client, key string, fields []string) ([]interface{}, error) {
    if key == "hmget_scenario1_key" {
        str := []string{"111", "222", "333"}
        counters := getCountersForInterfaces(str)
        return counters, nil
    } else if key == "hmget_scenario2_key" {
        return nil, errors.New("HmGet scenario2_key error")
    }
    return nil, nil
}

func mockHGetAllFunction(redisClient *redis.Client, key string) (map[string]string, error) {
    if key == "hgetall_scenario1_key" {
        return getInterfaceToODIMapping(), nil
    } else if key == "hgetall_scenario2_key" {
        return (map[string]string)(nil), errors.New("HGetAll scenario2_key error")
    }
    return nil, nil
}

/* Test HGetAll returns successfuly. */
func Test_RedisProvider_HGetAllReturnsSuccessfuly(t *testing.T) {
    // Mock
    executeHmGet = mockHmGetFunction
    executeHGetAll = mockHGetAllFunction

    // Act
    redisProvider := RedisProvider{}
    result, _ := redisProvider.HGetAll(2, "hgetall_scenario1_key")

    // Assert
    assert := assert.New(t)
    assert.NotEqual(nil, result, "Result is expected to be non nil")
    assert.Equal(3, len(result), "GetInterfaceCounters: length of resulting map is expected to be 3")
    assert.Equal(oid1, result[ethernet1], "oid-1 expected for ethernet1")
}

/* Test HGetAll Returns error */
func Test_RedisProvider_HGetAllReturnsError(t *testing.T) {
    // Mock
    executeHmGet = mockHmGetFunction
    executeHGetAll = mockHGetAllFunction

    // Act
    redisProvider := RedisProvider{}
    result, _ := redisProvider.HGetAll(2, "hgetall_scenario2_key")

    // Assert
    assert := assert.New(t)
    assert.NotEqual(nil, result, "Result is expected to be non nil")
}

/* Test HGetAll returns error for invalid database id */
func Test_RedisProvider_HGetAllReturnsError_ForInvalidDatabaseId(t *testing.T) {
    // Mock
    executeHmGet = mockHmGetFunction
    executeHGetAll = mockHGetAllFunction

    // Act
    redisProvider := RedisProvider{}
    result, err := redisProvider.HGetAll(20, "any_key")

    // Assert
    assert := assert.New(t)
    assert.Equal((map[string]string)(nil), result, "Result is expected to be non nil")
    assert.NotEqual(nil, err, "err is expected to be non-nil")
    result, err = redisProvider.HGetAll(21, "any_key")
    assert.Equal((map[string]string)(nil), result, "Result is expected to be non nil")
    assert.NotEqual(nil, err, "err is expected to be non-nil")
    result, err = redisProvider.HGetAll(22, "any_key")
    assert.Equal((map[string]string)(nil), result, "Result is expected to be non nil")
    assert.NotEqual(nil, err, "err is expected to be non-nil")
}

/* Test HmGet returns successfuly */
func Test_RedisProvider_HmGetReturnsSuccessfuly(t *testing.T) {
    // Mock
    executeHmGet = mockHmGetFunction
    executeHGetAll = mockHGetAllFunction

    // Act
    redisProvider := RedisProvider{}
    fields := []string{"key1", "key2", "key3"}
    result, err := redisProvider.HmGet(2, "hmget_scenario1_key", fields)

    // Assert
    assert := assert.New(t)
    assert.NotEqual(nil, result, "Result is expected to be non nil")
    assert.Equal(nil, err, "err is expected to be nil")
    assert.Equal(3, len(result), "GetInterfaceCounters: length of resulting map is expected to be 3")
    assert.Equal("111", result[0].(string), "key1 is expected to have 111")
    assert.Equal("222", result[1].(string), "key1 is expected to have 222")
    assert.Equal("333", result[2].(string), "key1 is expected to have 333")
}

/* Test HmGet returns error */
func Test_RedisProvider_HmGetReturnsError(t *testing.T) {

    // Mock
    executeHmGet = mockHmGetFunction
    executeHGetAll = mockHGetAllFunction

    // Act
    redisProvider := RedisProvider{}
    fields := []string{"key1", "key2", "key3"}
    result, err := redisProvider.HmGet(2, "hmget_scenario2_key", fields)

    // Assert
    assert := assert.New(t)
    assert.Equal(([]interface{})(nil), result, "Result is expected to be nil")
    assert.NotEqual(nil, err, "err is expected to be non-nil")
}

/* Test HmGet returns error for invalid database id */
func Test_RedisProvider_HmGetReturnsError_ForInvalidDatabaseId(t *testing.T) {
    // Mock
    executeHmGet = mockHmGetFunction
    executeHGetAll = mockHGetAllFunction

    // Act
    redisProvider := RedisProvider{}
    fields := []string{"key1", "key2", "key3"}
    result, err := redisProvider.HmGet(20, "any_key", fields)

    // Assert
    assert := assert.New(t)
    assert.Equal(([]interface{})(nil), result, "Result is expected to be nil")
    assert.NotEqual(nil, err, "err is expected to be non-nil")
    result, err = redisProvider.HmGet(21, "any_key", fields)
    assert.Equal(([]interface{})(nil), result, "Result is expected to be nil")
    assert.NotEqual(nil, err, "err is expected to be non-nil")
           result, err = redisProvider.HmGet(22, "any_key", fields)
    assert.Equal(([]interface{})(nil), result, "Result is expected to be nil")
    assert.NotEqual(nil, err, "err is expected to be non-nil")
}

/* Test GetRedisConnectionForDatabase returns connection successfuly */
func Test_GetRedisConnectionForDatabase_ReturnsConnectionSuccessfuly(t *testing.T) {
    // Mock
    dbToRedisClientMapping = make(map[int]*redis.Client)

    // Act
    redisClient1, _ := GetRedisConnectionForDatabase(2)
    redisClient2, _ := GetRedisConnectionForDatabase(2)

    // Assert
    assert := assert.New(t)
    assert.NotEqual(nil, redisClient1, "redisClient1 is expected to be non nil")
    assert.NotEqual(nil, redisClient2, "redisClient2 is expected to be non nil")
    if !(redisClient1 == redisClient2) {
        t.Errorf("RedisClient1 and RedisClient2 is expected point to same redis client.")
    }
}

/* Test hmGetFunction and hGetAllFunction function */
func Test_hmGetFunction_hGetAllFunction(t *testing.T) {

    redisServer := mockRedis()
    redisClient := redis.NewClient(&redis.Options{
        Addr: redisServer.Addr(),
    })

    value := map[string]interface{}{"a": "1", "b": "2", "c": "3"}
    redisClient.HMSet("abc", value)

    args := []string{"a", "b"}
    result, err := hmGetFunction(redisClient, "abc", args)
    assert := assert.New(t)
    assert.NotEqual(nil, result, "result expected to be non nil")
    assert.Equal(nil, err, "err expected to be nil")
    assert.Equal("1", result[0].(string), "result[0] is exptected to be 1")
    assert.Equal("2", result[1].(string), "result[1] is exptected to be 2")

    resultMap, error := hGetAllFunction(redisClient, "abc")
    assert.NotEqual(nil, resultMap, "resultMap expected to be non nil")
    assert.Equal(nil, error, "error expected to be nil")
    assert.Equal("1", resultMap["a"], "resultMap[a] is exptected to be 1")
    assert.Equal("2", resultMap["b"], "resultMap[b] is exptected to be 2")
    assert.Equal("3", resultMap["c"], "resultMap[c] is exptected to be 2")
}

func mockRedis() *miniredis.Miniredis {
    server, err := miniredis.Run()
    if err != nil {
        panic(err)
    }
    return server
}
