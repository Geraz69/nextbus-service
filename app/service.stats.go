package main

import "github.com/emicklei/go-restful"
import "strconv"
import "strings"
import "math"
import "time"
import "fmt"
import "log"

type StatsCounter struct {
	Incrementer
}

type Hits struct {
	Endpoint    string
	NumRequests int
}

type Times struct {
	NumRequests int
	MoreThan    string
	LessThan    string
}

func bootstrapStatsService(ws *restful.WebService, incr Incrementer) {
	statsCounter := StatsCounter{incr}
	ws.Filter(statsCounter.countAndMeasureTime)
	ws.Route(ws.GET("/stats/hits").To(statsCounter.hits))
	ws.Route(ws.GET("/stats/times").To(statsCounter.times))
}

func (counter StatsCounter) countAndMeasureTime(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	now := time.Now()
	chain.ProcessFilter(req, resp)
	taken := time.Now().Sub(now).Nanoseconds()
	logarithm := int(math.Log10(float64(taken)))
	if err := counter.Incr(fmt.Sprintf("hits:%v", req.Request.URL)); err != nil {
		log.Print(err.Error())
	}
	if err := counter.Incr(fmt.Sprintf("time:%v", logarithm)); err != nil {
		log.Print(err.Error())
	}
}

func (counter StatsCounter) hits(req *restful.Request, resp *restful.Response) {
	hits := []Hits{}
	keys, err := counter.Iter()
	if err != nil {
		respond(resp, nil, err)
	}
	for key := range keys {
		if strings.HasPrefix(key, "hits:") {
			value, found, err := counter.Get(key)
			if err != nil {
				respond(resp, nil, err)
				return
			}
			if found {
				hits = append(hits, Hits{key[5:], value})
			}
		}
	}
	respond(resp, hits, nil)
}

func (counter StatsCounter) times(req *restful.Request, resp *restful.Response) {
	times := []Times{}
	keys, err := counter.Iter()
	if err != nil {
		respond(resp, nil, err)
	}
	for key := range keys {
		if strings.HasPrefix(key, "time:") {
			value, found, err := counter.Get(key)
			if err != nil {
				respond(resp, nil, err)
				return
			}
			if found {
				logarithm, _ := strconv.Atoi(key[5:])
				lessThan := int(math.Pow10(logarithm))
				moreThan := int(math.Pow10(logarithm - 1))
				times = append(times, Times{value, time.Duration(moreThan).String(), time.Duration(lessThan).String()})
			}
		}
	}
	respond(resp, times, nil)
}

func respond(resp *restful.Response, entity interface{}, err error) {
	if err != nil {
		switch err.(type) {
		case *time.ParseError:
			resp.WriteErrorString(400, "400: Bad Request")
		case *strconv.NumError:
			resp.WriteErrorString(400, "400: Bad Request")
		default:
			log.Print(err.Error())
			resp.WriteErrorString(500, "500: Internal Server Error")
		}
	} else {
		if entity != nil {
			resp.WriteEntity(entity)
		} else {
			resp.WriteErrorString(404, "404: Page Not Found")
		}
	}
}
