package main

import "github.com/geraz69/nextbus"
import "strconv"
import "log"

type NextBus struct {
	Cacher
}

type RoutesAvailability struct {
	Running    []nextbus.Route
	NotRunning []nextbus.Route
	Unknown    []nextbus.Route
}

type SchedulesRange struct {
	start int
	end   int
}

func (nb NextBus) GetAgencies() ([]nextbus.Agency, error) {
	cacheValueKey := "agencies"
	lockId, err := nb.Lock(cacheValueKey)
	if err != nil {
		return nil, err
	}
	defer nb.Unlock(cacheValueKey, lockId)
	value := []nextbus.Agency{}
	found, err := nb.Get(cacheValueKey, &value)
	if err != nil {
		return nil, err
	}
	if !found {
		log.Print("Fetching agencies")
		value, err = nextbus.GetAgencies()
		if err != nil {
			return nil, err
		}
		if value != nil {
			nb.Set(cacheValueKey, value)
		}
	}
	return value, nil
}

func (nb NextBus) GetAgency(agencyTag string) (*nextbus.Agency, error) {
	agencies, err := nb.GetAgencies()
	for _, agency := range agencies {
		if agency.Tag == agencyTag {
			return &agency, nil
		}
	}
	return nil, err
}

func (nb NextBus) GetRoutes(agencyTag string) ([]nextbus.Route, error) {
	cacheValueKey := "agencies/" + agencyTag + "/routes"
	lockId, err := nb.Lock(cacheValueKey)
	if err != nil {
		return nil, err
	}
	defer nb.Unlock(cacheValueKey, lockId)
	value := []nextbus.Route{}
	found, err := nb.Get(cacheValueKey, &value)
	if err != nil {
		return nil, err
	}
	if !found {
		log.Printf("Fetching routes for agency: %v", agencyTag)
		value, err = nextbus.GetRoutes(agencyTag)
		if err != nil {
			return nil, err
		}
		if value != nil {
			nb.Set(cacheValueKey, value)
		}
	}
	return value, nil
}

func (nb NextBus) GetRoute(agencyTag, routeTag string) (*nextbus.Route, error) {
	routes, err := nb.GetRoutes(agencyTag)
	for _, route := range routes {
		if route.Tag == routeTag {
			return &route, nil
		}
	}
	return nil, err
}

func (nb NextBus) GetStops(agencyTag, routeTag string) ([]nextbus.Stop, error) {
	cacheValueKey := "agencies/" + agencyTag + "/routes/" + routeTag + "/config"
	lockId, err := nb.Lock(cacheValueKey)
	if err != nil {
		return nil, err
	}
	defer nb.Unlock(cacheValueKey, lockId)
	value := &nextbus.RouteConfig{}
	found, err := nb.Get(cacheValueKey, &value)
	if err != nil {
		return nil, err
	}
	if !found {
		log.Printf("Fetching stops for agency/route: %v/%v", agencyTag, routeTag)
		*value, err = nextbus.GetRouteConfig(agencyTag, routeTag, true, true)
		if err != nil {
			return nil, err
		}
		if value != nil {
			nb.Set(cacheValueKey, value)
		}
	}
	return value.Stop, nil
}

func (nb NextBus) GetStop(agencyTag, routeTag, stopTag string) (*nextbus.Stop, error) {
	stops, err := nb.GetStops(agencyTag, routeTag)
	for _, stop := range stops {
		if stop.Tag == stopTag {
			return &stop, nil
		}
	}
	return nil, err
}

func (nb NextBus) GetPredictions(agencyTag, routeTag, stopTag string) ([]nextbus.Prediction, error) {
	cacheValueKey := "agencies/" + agencyTag + "/routes/" + routeTag + "/stops/" + stopTag + "/predictions"
	lockId, err := nb.Lock(cacheValueKey)
	if err != nil {
		return nil, err
	}
	defer nb.Unlock(cacheValueKey, lockId)
	value := &nextbus.Predictions{}
	found, err := nb.Get(cacheValueKey, &value)
	if err != nil {
		return nil, err
	}
	if !found {
		log.Printf("Fetching predictions for agency/route/stop: %v/%v/%v", agencyTag, routeTag, stopTag)
		*value, err = nextbus.GetPredictions(agencyTag, routeTag, stopTag)
		if err != nil {
			return nil, err
		}
		if value != nil {
			nb.Set(cacheValueKey, value)
		}
	}
	direction := value.Direction
	return direction.Prediction, nil
}

func (nb NextBus) GetSchedules(agencyTag, routeTag string) ([]nextbus.Schedule, error) {
	cacheValueKey := "agencies/" + agencyTag + "/routes/" + routeTag + "/schedules"
	lockId, err := nb.Lock(cacheValueKey)
	if err != nil {
		return nil, err
	}
	defer nb.Unlock(cacheValueKey, lockId)
	value := []nextbus.Schedule{}
	found, err := nb.Get(cacheValueKey, &value)
	if err != nil {
		return nil, err
	}
	if !found {
		log.Printf("Fetching schedules for agency/route: %v/%v", agencyTag, routeTag)
		value, err = nextbus.GetSchedules(agencyTag, routeTag)
		if err != nil {
			return nil, err
		}
		if value != nil {
			nb.Set(cacheValueKey, value)
		}
	}
	return value, nil
}

func (nb NextBus) GetSchedulesRange(agencyTag, routeTag string) (*SchedulesRange, error) {
	// TODO: delimit the schedules to compare using Schedule.ServiceClass and Schedule.Direction
	// i.e. sat:Inbound
	start, end := 1<<63-1, -1<<63 //max and min ints
	schedules, err := nb.GetSchedules(agencyTag, routeTag)
	if err != nil {
		return nil, err
	}
	for _, schedule := range schedules {
		for _, tr := range schedule.Tr {
			for _, stop := range tr.Stop {
				if stop.Content != "--" {
					epoch, err := strconv.Atoi(stop.EpochTime)
					if err != nil {
						return nil, err
					}
					if start > epoch {
						start = epoch
					}
					if epoch > end {
						end = epoch
					}
				}
			}
		}
	}
	if start > end {
		return nil, nil
	}
	return &SchedulesRange{start, end}, nil
}

func (nb NextBus) GetRoutesAvailability(agencyTag string, time int) (*RoutesAvailability, error) {
	// timeNextDay is the same time pluse the number of seconds in a day.
	// is used to determine if the route is running when the range overlaps more than one day.
	timeNextDay := time + 24*60*60*1000
	routes, err := nb.GetRoutes(agencyTag)
	if err != nil {
		return nil, err
	}
	running := []nextbus.Route{}
	notRunning := []nextbus.Route{}
	unavailableData := []nextbus.Route{}
	for _, route := range routes {
		schedulesRange, err := nb.GetSchedulesRange(agencyTag, route.Tag)
		if err != nil || schedulesRange == nil {
			log.Printf("Schedules for route <%v> either failed or returned an empty result", route.Tag)
			unavailableData = append(unavailableData, route)
		} else {
			if time >= schedulesRange.start && time <= schedulesRange.end ||
				timeNextDay >= schedulesRange.start && timeNextDay <= schedulesRange.end {
				running = append(running, route)
			} else {
				notRunning = append(notRunning, route)
			}
		}
	}
	return &RoutesAvailability{running, notRunning, unavailableData}, nil
}
