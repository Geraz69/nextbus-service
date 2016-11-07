package main

import "github.com/geraz69/lru"
import "encoding/json"
import "math/rand"
import "errors"
import "bytes"
import "sync"
import "time"

type InProcessCache struct {
	data    lru.LRU
	lock    lru.LRU
	ttlData time.Duration
	ttlLock time.Duration
	mutex   sync.Mutex
}

type InProcessCounter struct {
	lruCounter lru.LRUCounter
}

func NewInProcessCache(capacity int, ttlData time.Duration, ttlLock time.Duration) InProcessCache {
	return InProcessCache{
		data:    *lru.New(nil, nil, capacity, ttlData),
		lock:    *lru.New(nil, nil, capacity, ttlLock),
		mutex:   sync.Mutex{},
		ttlData: ttlData,
		ttlLock: ttlLock,
	}
}

func (cache InProcessCache) Get(key string, v interface{}) (bool, error) {
	b, ok := cache.data.Get(lru.Key(key))
	if b == nil || !ok {
		return false, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(b.([]byte)))
	err := decoder.Decode(&v)
	return true, err
}

func (cache InProcessCache) Set(key string, value interface{}) error {
	if value == nil {
		panic("value shouldn't be nil")
	}
	b := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(b)
	err := encoder.Encode(value)
	if err == nil {
		cache.data.Set(lru.Key(key), lru.Value(b.Bytes()))
	}
	return err
}

func (cache InProcessCache) Lock(key string) (int, error) {
	lockId := rand.Int()
	now := time.Now()
	expiration := cache.ttlData.Nanoseconds() / 10
	for taken := int64(0); taken < expiration; taken = time.Now().Sub(now).Nanoseconds() {
		cache.mutex.Lock()
		if existing, _ := cache.lock.Get(lru.Key(key)); existing == nil {
			cache.lock.Set(lru.Key(key), lockId)
			cache.mutex.Unlock()
			return lockId, nil
		}
		cache.mutex.Unlock()
		time.Sleep(time.Duration(cache.ttlLock.Nanoseconds() / 10))
	}
	return 0, errors.New("unable to aquire the log for key: " + key)
}

func (cache InProcessCache) Unlock(key string, lockId int) {
	cache.mutex.Lock()
	if existing, _ := cache.lock.Get(lru.Key(key)); existing != nil && existing.(int) == lockId {
		cache.lock.Set(lru.Key(key), nil)
	}
	cache.mutex.Unlock()
}

func NewInProcessCounter(capacity int) InProcessCounter {
	return InProcessCounter{*lru.NewLRUCounter(nil, capacity)}
}

func (counter InProcessCounter) Get(key string) (int, bool, error) {
	value, ok := counter.lruCounter.Get(lru.Key(key))
	return int(value), ok, nil
}

func (counter InProcessCounter) Incr(key string) error {
	counter.lruCounter.Incr(lru.Key(key), 1)
	return nil
}

func (counter InProcessCounter) Iter() (chan string, error) {
	dataLen := counter.lruCounter.Len()
	lruKeys, lruValues := make(chan lru.Key, dataLen), make(chan lru.Value, dataLen)
	counter.lruCounter.Iter(lruKeys, lruValues)
	keys := make(chan string, dataLen)
	for k := range lruKeys {
		keys <- k.(string)
	}
	close(keys)
	return keys, nil
}
