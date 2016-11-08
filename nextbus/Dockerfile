FROM golang:latest

# Install our dependencies
RUN go get github.com/geraz69/lru
RUN go get github.com/emicklei/go-restful
RUN go get github.com/pelletier/go-toml
RUN go get github.com/geraz69/nextbus
RUN go get github.com/garyburd/redigo/redis

# Copy our sources
ADD . /go/src/github.com/geraz69/nextbus-service/nextbus

# Install api binary globally within container 
RUN go install github.com/geraz69/nextbus-service/nextbus

# Set binary as entrypoint
ENTRYPOINT /go/bin/nextbus /go/src/github.com/geraz69/nextbus-service/nextbus/config.toml

# Expose default port (8080)
EXPOSE 8080 
