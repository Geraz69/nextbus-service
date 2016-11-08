# NextBus Service

By [Gerardo Garcia Mendez](https://twitter.com/Geraz69).

Simple [Go](https://golang.org/) wrapper for the [NextBus](http://www.nextbus.com/xmlFeedDocs/NextBusXMLFeed.pdf) public XML feed. It exposes a series of RESTful endpoints that translate to NextBus service commands, and has the following characteristics:
* Queries to the NextBus feed are throttled so that the same command is never executed more than once with the same parameters before a configurable amount of time (defaults to 2 minutes, see configuration).
* Responses are cached using one of two different providers: a Redis server or an in-process LRU cache.
* The app is stateless, so you can create and destroy multiple instances without worrying about coordination. All coordination is done thru some pessimistic locking in the shared cache when the provider is Redis. Locks also expire after a configurable amount of time.
* The app also exposes some API endpoints for statistics about the usage of the service.
* All responses are written on JSON.

Next bus service has some caveats that need to be taken care when implementing a client application.

* The service has a restricted data transfer rate. When the rate limit is reached it returns null for the queries made until enough time has passed and the rate replenished. If the cache is not warm the data heavy endpoints might return null results.
* Some of the commands return malformed data. For example, the specification for predictions command says that it will return a list of predictions, like `predictions: [ {prediction1}, {prediction2}, ...]`. But when the result is only one element long the response structure changes, instead of doing a one item array, like `predictions: [ {prediction1} ]`, it skips the array and delivers the object without a wrapper, like `predictions: {prediction}`.

## Run the application locally

To run this in your machine you needs to have Go 1.6 or higher, you can download it from [here](https://golang.org/dl/). If you happen to use Mac OSX you can use [Homebrew](http://brew.sh/) to install it:

```bash
$ brew install golang
```
Either way you also will need to setup your go environment ($GOPATH variable), you can check how to do it [here](https://golang.org/doc/install).

You will need to get this repo and its dependencies:
```bash
# Main repo
$ go get github.com/geraz69/nextbus-service
# Dependencies
$ go get github.com/geraz69/lru
$ go get github.com/geraz69/nextbus
$ go get github.com/emicklei/go-restful
$ go get github.com/garyburd/redigo/redis
$ go get github.com/pelletier/go-toml
```

From there you can run the code directly without installing. You can also install and create a single binary file.
```bash
# Run in place with default config
$ cd $GOPATH/src/github.com/geraz69/nextbus-service
$ go run *.go

# Install and run binary
$ go install github.com/geraz69/nextbus-service
$ $GOPATH/src/bin/nextbus-service /path/to/config.toml
```
Now go and curl/wget or browse you localhost on default config port.
## Config

The first argument of the program is an optional parameter providing the location of the config file. The file is in [TOML](https://github.com/toml-lang/toml) format, a superset of JSON. If none is provided it tries to load config.toml in the repo. Program will panic if it fails to load a config file. All the parameters inside the config are mandatory.

Please open config.toml to see the available configurations and their defaults.

## API endpoints

* `/api/v1/agencies` Lists all the existing agencies.
* `/api/agencies/{agency}` Shows one agency based on agency tag.
* `/api/agencies/{agency}/routes` Lists all the routes for a given agency.
* `/api/agencies/{agency}/routes/{route}` Shows one route for a given agency based on a route tag.
* `/api/agencies/{agency}/routes/{route}/stops` Lists all the stops for a given route in an agency.
* `/api/agencies/{agency}/routes/{route}/stops/{stop}` Shows the info for a stop based on a stop tag, given a route and an agency tags as well.
* `/api/agencies/{agency}/routes/{route}/stops/{stop}/predictions` Retrieves the predictions related to a stop. Predictions are real time, so if the api end point is queried for a stop in a route that has finished its runs for the day then it will turn out as null (refer to NextBus docs for more details on predictions).
* `/api/agencies/{agency}/routes/{route}/schedules` Retrieves the schedules of a route. Its a matrix consisting of the stops in a route and the different runs though that route. The intersection of those is the time at which a given run of the route will go by a given stop (route 81X and K_OWL of sf-muni agency are know to always fail due to malformed responses).
* `/api/agencies/{agency}/routes/availability?time=<time>` Retrieves the general availability for all the routes in an agency for a given time during the day. The response is divided in three lists of route objects: available, unavailable and unknown. Available routes are the ones that will be performing runs at the specified time. Unavailable are the routes that have already finished or haven't started their runs for the day. Unknown are the routes that were queried but which response from the NextBus service wasn't successful, either because of data transfer rate limiting or because of malformed data in the response. The time parameter is optional and defaults to the current time. If specified it should follow the next format: `hh`, `hh:mm` or `hh:mm:ss`. For this call most of the routes will fall under the unknown category if the cache has not been warmed (i.e. the first times the endpoint is called).
* `/api/stats/hits` This endpoint provides a list of the exposed APIs endpoints (including itself) and the numbers of hits each one has received (number of calls in a single execution of the program)
* `/api/stats/times` This endpoint provides a summary of the response time for each one of the calls made to all the endpoints, grouped by the logarithmic amount of time taken to fulfill the request.

## Running in distributed mode

With docker-compose you can run the application with multiple instances. A number of stateless instances will connect to a shared Redis server. All of them will serve requests in a round-robin fashion, balanced by an NGINX instance acting as a reverse proxy. The lock mechanism prevents calls to a single resource to happen more often than <ttlData> any time that the request finishes in less than <ttlLock> (see config).

Please install [docker-compose](https://docs.docker.com/compose/gettingstarted/). Again, you can use Homebrew if running on Mac OSX:

```bash
$ brew install docker-compose
# you probably also need docker and virtualbox
```
To bootstrap docker compose you can run:
```bash
$ docker-machine create --driver=virtualbox default #If not created before
$ docker-machine start default
$ eval $(docker-machine env default)
```
After docker machine is up and running you can run the program with:
```bash
docker-compose build
docker-compose up
# Set the number of containers running the application
docker-compose scale nextbus=3
```
To hit the api endpoint you will need to hit the docker machine IP address in the port 8080. You can use `docker-machine env default` to get that information.

# TODO
* Add proper unit testing
* ping and/or health api endpoint.
* Handling null responses better. Return a proper HTTP status accordingly
* Add a redundant Redis setup (master/master or master/slave). Though that will involve upgrading the lock strategy to something like this: http://redis.io/topics/distlock
* Do proper lower camel case in JSON responses.
* Manage go dependencies in a more concise way.
* Make the times endpoint have a configurable logarithm base, as oposed to a hardcoded 10.
