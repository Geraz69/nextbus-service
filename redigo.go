package main

import "github.com/garyburd/redigo/redis"
import "encoding/json"
import "math/rand"
import "errors"
import "bytes"
import "time"
import "fmt"

type RedisCache struct {
	url     string
	ttlData time.Duration
	ttlLock time.Duration
}

type RedisCounter struct {
	url string
}

const unlockScript = `
	if redis.call("get",KEYS[1]) == ARGV[1] then
	    return redis.call("del",KEYS[1])
	else
	    return 0
	end
`

func NewRedisCache(url string, ttlData time.Duration, ttlLock time.Duration) RedisCache {
	return RedisCache{
		url:     url,
		ttlData: ttlData,
		ttlLock: ttlLock,
	}
}

func (cache RedisCache) Get(key string, v interface{}) (bool, error) {
	conn, err := redis.Dial("tcp", cache.url)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	b, err := redis.Bytes(conn.Do("GET", key))
	if err != nil && err != redis.ErrNil {
		return false, err
	} else if err == redis.ErrNil {
		return false, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	err = decoder.Decode(&v)
	return true, err
}

func (cache RedisCache) Set(key string, value interface{}) error {
	if value == nil {
		panic("value shouldn't be nil")
	}
	conn, err := redis.Dial("tcp", cache.url)
	if err != nil {
		return err
	}
	defer conn.Close()
	b := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(b)
	err = encoder.Encode(&value)
	if err != nil {
		return err
	}
	_, err = conn.Do("SET", key, b, "EX", fmt.Sprintf("%v", cache.ttlData.Seconds()))
	return err
}

func (cache RedisCache) Lock(key string) (int, error) {
	conn, err := redis.Dial("tcp", cache.url)
	defer conn.Close()
	if err != nil {
		return 0, err
	}
	lockId := rand.Int()
	now := time.Now()
	expiration := cache.ttlData.Nanoseconds() / 10
	for taken := int64(0); taken < expiration; taken = time.Now().Sub(now).Nanoseconds() {
		if res, err := conn.Do("SET", "lock:"+key, lockId, "EX", fmt.Sprintf("%v", cache.ttlLock.Seconds())); err != nil {
			return 0, err
		} else if res == "OK" {
			return lockId, nil
		}
		time.Sleep(time.Duration(cache.ttlLock / 10))
	}
	return 0, errors.New("unable to aquire the log for key: " + key)
}

func (cache RedisCache) Unlock(key string, lockId int) {
	conn, err := redis.Dial("tcp", cache.url)
	defer conn.Close()
	if err == nil {
		cmd := redis.NewScript(1, unlockScript)
		redis.Int(cmd.Do(conn, key, lockId))
	}
}

func NewRedisCounter(url string) RedisCounter {
	return RedisCounter{url}
}

func (counter RedisCounter) Get(key string) (int, bool, error) {
	conn, err := redis.Dial("tcp", counter.url)
	defer conn.Close()
	if err != nil {
		return 0, false, err
	}
	count, err := redis.Int(conn.Do("GET", key))
	if err != nil && err != redis.ErrNil {
		return 0, false, err
	} else if err == redis.ErrNil {
		return 0, false, nil
	}
	return int(count), true, nil
}

func (counter RedisCounter) Incr(key string) error {
	conn, err := redis.Dial("tcp", counter.url)
	defer conn.Close()
	if err != nil {
		return err
	}
	_, err = conn.Do("INCR", key)
	return err
}

func (counter RedisCounter) Iter() (chan string, error) {
	conn, err := redis.Dial("tcp", counter.url)
	defer conn.Close()
	values, err := redis.Strings(conn.Do("KEYS", "*"))
	if err != nil {
		return nil, err
	}
	keys := make(chan string, len(values))
	for x := 0; x < len(values); x++ {
		keys <- values[x]
	}
	close(keys)
	return keys, nil
}
