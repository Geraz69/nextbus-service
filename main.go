package main

import "github.com/emicklei/go-restful"
import "github.com/pelletier/go-toml"
import "net/http"
import "errors"
import "time"
import "fmt"
import "log"
import "os"

type Cacher interface {
	Get(key string, value interface{}) (found bool, err error)
	Set(key string, value interface{}) (err error)
	Lock(key string) (int, error)
	Unlock(key string, lockId int)
}

type Incrementer interface {
	Get(key string) (v int, found bool, err error)
	Incr(key string) (err error)
	Iter() (chan string, error)
}

func main() {
	var err error
	defer func() {
		if err != nil {
			log.Println("Error:", err.Error())
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	}()

	configPath := "config.toml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	config, err := toml.LoadFile(configPath)
	if err != nil {
		err = errors.New("Unable to load config file: " + err.Error())
		return
	}

	var cache Cacher
	var incr Incrementer

	provider := config.Get("cache.provider")

	ttlData, err := time.ParseDuration(config.Get("cache.ttlData").(string))
	if err != nil {
		err = errors.New("Unable to read ttlData: " + err.Error())
		return
	}

	ttlLock, err := time.ParseDuration(config.Get("cache.ttlLock").(string))
	if err != nil {
		err = errors.New("Unable to read ttlLock: " + err.Error())
		return
	}

	switch provider {
	case "lru":
		capacity := config.Get("lru.capacity").(int64)
		cache = NewInProcessCache(int(capacity), ttlData, ttlLock)
		incr = NewInProcessCounter(int(capacity))
	case "redis":
		url := config.Get("redis.url").(string)
		cache = NewRedisCache(url, ttlData, ttlLock)
		incr = NewRedisCounter(url)
	default:
		err = errors.New("unknown or unspecified cache provider")
		return
	}

	ws := new(restful.WebService)
	ws.Path("/api").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	bootstrapNextBusService(ws, cache)
	bootstrapStatsService(ws, incr)

	restful.Add(ws)

	port := config.Get("service.port").(int64)
	http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
}
