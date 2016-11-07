package main

import "github.com/emicklei/go-restful"
import "time"
import "fmt"

func bootstrapNextBusService(ws *restful.WebService, cache Cacher) {
	nextBus := NextBus{cache}
	ws.Route(ws.GET("/agencies").To(nextBus.agencies))
	ws.Route(ws.GET("/agencies/{agency}").To(nextBus.agency))
	ws.Route(ws.GET("/agencies/{agency}/routes").To(nextBus.routes))
	ws.Route(ws.GET("/agencies/{agency}/routes/{route}").To(nextBus.route))
	ws.Route(ws.GET("/agencies/{agency}/routes/{route}/stops").To(nextBus.stops))
	ws.Route(ws.GET("/agencies/{agency}/routes/{route}/stops/{stop}").To(nextBus.stop))
	ws.Route(ws.GET("/agencies/{agency}/routes/{route}/stops/{stop}/predictions").To(nextBus.predictions))
	ws.Route(ws.GET("/agencies/{agency}/routes/{route}/schedules").To(nextBus.schedules))

	ws.Route(ws.GET("/agencies/{agency}/routes/availability").To(nextBus.routesAvailability))
}

func (nb *NextBus) agencies(req *restful.Request, resp *restful.Response) {
	agencies, err := nb.GetAgencies()
	respond(resp, agencies, err)
}

func (nb *NextBus) agency(req *restful.Request, resp *restful.Response) {
	agencyTag := req.PathParameter("agency")
	agency, err := nb.GetAgency(agencyTag)
	respond(resp, agency, err)
}

func (nb *NextBus) routes(req *restful.Request, resp *restful.Response) {
	agencyTag := req.PathParameter("agency")
	routes, err := nb.GetRoutes(agencyTag)
	respond(resp, routes, err)
}

func (nb *NextBus) route(req *restful.Request, resp *restful.Response) {
	agencyTag := req.PathParameter("agency")
	routeTag := req.PathParameter("route")
	route, err := nb.GetRoute(agencyTag, routeTag)
	respond(resp, route, err)
}

func (nb *NextBus) stops(req *restful.Request, resp *restful.Response) {
	agencyTag := req.PathParameter("agency")
	routeTag := req.PathParameter("route")
	route, err := nb.GetStops(agencyTag, routeTag)
	respond(resp, route, err)
}

func (nb *NextBus) stop(req *restful.Request, resp *restful.Response) {
	agencyTag := req.PathParameter("agency")
	routeTag := req.PathParameter("route")
	stopTag := req.PathParameter("stop")
	stop, err := nb.GetStop(agencyTag, routeTag, stopTag)
	respond(resp, stop, err)
}

func (nb *NextBus) predictions(req *restful.Request, resp *restful.Response) {
	agencyTag := req.PathParameter("agency")
	routeTag := req.PathParameter("route")
	stopTag := req.PathParameter("stop")
	predictions, err := nb.GetPredictions(agencyTag, routeTag, stopTag)
	respond(resp, predictions, err)
}

func (nb *NextBus) schedules(req *restful.Request, resp *restful.Response) {
	agencyTag := req.PathParameter("agency")
	routeTag := req.PathParameter("route")
	schedules, err := nb.GetSchedules(agencyTag, routeTag)
	respond(resp, schedules, err)
}

func (nb *NextBus) routesAvailability(req *restful.Request, resp *restful.Response) {
	agencyTag := req.PathParameter("agency")
	timeStr := req.QueryParameter("time")
	switch len(timeStr) {
	case 0:
		timeStr = time.Now().Format("15:04:05")
	case 1:
		timeStr = fmt.Sprintf("0%v:00:00", timeStr)
	case 2:
		timeStr = fmt.Sprintf("%v:00:00", timeStr)
	case 4:
		timeStr = fmt.Sprintf("0%v:00", timeStr)
	case 5:
		timeStr = fmt.Sprintf("%v:00", timeStr)
	}
	time, err := time.Parse("2006-01-03 15:04:05", "1970-01-01 "+timeStr)
	if err != nil {
		respond(resp, nil, err)
		return
	}
	availability, err := nb.GetRoutesAvailability(agencyTag, int(time.Unix()*1000))
	respond(resp, availability, err)
}
