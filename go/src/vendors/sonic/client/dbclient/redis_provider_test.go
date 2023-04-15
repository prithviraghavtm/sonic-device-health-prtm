package dbclient

import (
	"errors"
	"testing"

	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
)

/*
func init() {
	client := redis.NewClient(&redis.Options{
		Addr: ":6379",
	})
	dbToRedisClientMapping[2] = client
}
*/

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

func Test_RedisProvider_HGetAllReturnsSuccessfuly(t *testing.T) {

	executeHmGet = mockHmGetFunction
	executeHGetAll = mockHGetAllFunction

	redisProvider := RedisProvider{}
	result, _ := redisProvider.HGetAll(2, "hgetall_scenario1_key")
	assert := assert.New(t)
	assert.NotEqual(nil, result, "Result is expected to be non nil")
	assert.Equal(3, len(result), "GetInterfaceCounters: length of resulting map is expected to be 3")
	assert.Equal(oid1, result[ethernet1], "oid-1 expected for ethernet1")
}

func Test_RedisProvider_HGetAllReturnsError(t *testing.T) {

	executeHmGet = mockHmGetFunction
	executeHGetAll = mockHGetAllFunction

	redisProvider := RedisProvider{}
	result, _ := redisProvider.HGetAll(2, "hgetall_scenario2_key")
	assert := assert.New(t)
	assert.NotEqual(nil, result, "Result is expected to be non nil")
}

func Test_RedisProvider_HGetAllReturnsError_ForInvalidDatabaseId(t *testing.T) {

	executeHmGet = mockHmGetFunction
	executeHGetAll = mockHGetAllFunction

	redisProvider := RedisProvider{}
	result, err := redisProvider.HGetAll(20, "any_key")
	assert := assert.New(t)
	assert.Equal((map[string]string)(nil), result, "Result is expected to be non nil")
	assert.NotEqual(nil, err, "err is expected to be non-nil")
}

func Test_RedisProvider_HmGetReturnsSuccessfuly(t *testing.T) {

	executeHmGet = mockHmGetFunction
	executeHGetAll = mockHGetAllFunction

	redisProvider := RedisProvider{}
	fields := []string{"key1", "key2", "key3"}
	result, err := redisProvider.HmGet(2, "hmget_scenario1_key", fields)
	assert := assert.New(t)
	assert.NotEqual(nil, result, "Result is expected to be non nil")
	assert.Equal(nil, err, "err is expected to be nil")
	assert.Equal(3, len(result), "GetInterfaceCounters: length of resulting map is expected to be 3")
	assert.Equal("111", result[0].(string), "key1 is expected to have 111")
	assert.Equal("222", result[1].(string), "key1 is expected to have 222")
	assert.Equal("333", result[2].(string), "key1 is expected to have 333")
}

func Test_RedisProvider_HmGetReturnsError(t *testing.T) {

	executeHmGet = mockHmGetFunction
	executeHGetAll = mockHGetAllFunction

	redisProvider := RedisProvider{}
	fields := []string{"key1", "key2", "key3"}
	result, err := redisProvider.HmGet(2, "hmget_scenario2_key", fields)
	assert := assert.New(t)
	assert.Equal(([]interface{})(nil), result, "Result is expected to be nil")
	assert.NotEqual(nil, err, "err is expected to be non-nil")
}

func Test_RedisProvider_HmGetReturnsError_ForInvalidDatabaseId(t *testing.T) {

	executeHmGet = mockHmGetFunction
	executeHGetAll = mockHGetAllFunction

	redisProvider := RedisProvider{}
	fields := []string{"key1", "key2", "key3"}
	result, err := redisProvider.HmGet(20, "any_key", fields)
	assert := assert.New(t)
	assert.Equal(([]interface{})(nil), result, "Result is expected to be nil")
	assert.NotEqual(nil, err, "err is expected to be non-nil")
}

func Test_GetRedisConnectionForDatabase_ReturnsConnectionSuccessfuly(t *testing.T) {
	redisClient1, _ := GetRedisConnectionForDatabase(2)
	redisClient2, _ := GetRedisConnectionForDatabase(2)
	if !(redisClient1 == redisClient2) {
		t.Errorf("RedisClient1 and RedisClient2 is expected point to same redis client.")
	}
}

