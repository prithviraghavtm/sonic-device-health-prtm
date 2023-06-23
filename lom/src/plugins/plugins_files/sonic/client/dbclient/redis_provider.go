package dbclient

import (
    "errors"
    "fmt"
    "github.com/go-redis/redis"
)

const (
    redisServerAddr     = "localhost:6379"
    redisServerPassword = ""
    APPL_DB_ID          = 0
    COUNTER_DB_ID       = 2
)

var dbToRedisClientMapping map[int]*redis.Client

func init() {
    dbToRedisClientMapping = make(map[int]*redis.Client)
}

/*
Gets redis connection for a specific redis database. The connection is re-used.
If multiple go routines call this at same time, the last client will be stored.
*/
func GetRedisConnectionForDatabase(databaseIdentifier int) (*redis.Client, error) {
    if !validateDbIdentifier(databaseIdentifier) {
        return nil, errors.New(fmt.Sprintf("invalid databaseIdentifier (%d)", databaseIdentifier))
    }

    redisClient, ok := dbToRedisClientMapping[databaseIdentifier]

    if ok {
        return redisClient, nil
    }

    var client = redis.NewClient(&redis.Options{
        Addr:     redisServerAddr,
        Password: redisServerPassword,
        DB:       databaseIdentifier,
    })

    if client == nil {
        return nil, errors.New(fmt.Sprintf("client expected to be non nil for databaseIdentifier (%d)", databaseIdentifier))
    }

    dbToRedisClientMapping[databaseIdentifier] = client
    return client, nil
}

/* Validates if databaseIdentifier is allowed to be accessed by LoM" */
func validateDbIdentifier(databaseIdentifier int) bool {
    return databaseIdentifier == APPL_DB_ID || databaseIdentifier == COUNTER_DB_ID
}

type RedisProviderInterface interface {
    HmGet(database int, key string, fields []string) ([]interface{}, error)
    HGetAll(database int, key string) (map[string]string, error)
}

type RedisProvider struct {
}

func (redisProvider *RedisProvider) HmGet(database int, key string, fields []string) ([]interface{}, error) {
    client, err := GetRedisConnectionForDatabase(database)
    if err != nil {
        return nil, err
    }
    return executeHmGet(client, key, fields)
}

func (redisProvider *RedisProvider) HGetAll(database int, key string) (map[string]string, error) {
    client, err := GetRedisConnectionForDatabase(database)
    if err != nil {
        return nil, err
    }
    return executeHGetAll(client, key)
}

func hmGetFunction(redisClient *redis.Client, key string, fields []string) ([]interface{}, error) {
    return redisClient.HMGet(key, fields...).Result()
}

func hGetAllFunction(redisClient *redis.Client, key string) (map[string]string, error) {
    return redisClient.HGetAll(key).Result()
}

var executeHmGet = hmGetFunction
var executeHGetAll = hGetAllFunction
