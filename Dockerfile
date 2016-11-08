FROM golang:latest

# Install our dependencies
RUN go get github.com/geraz69/lru
RUN go get github.com/emicklei/go-restful
RUN go get github.com/pelletier/go-toml
RUN go get github.com/geraz69/nextbus
RUN go get github.com/garyburd/redigo/redis

# Copy our sources
ADD . /go/src/github.com/geraz69/nextbus-service

# Install api binary globally within container 
RUN go install github.com/geraz69/nextbus-service

# Copy our config
COPY ./config/nextbus-service/config.toml /etc/nextbus-service/config.toml

# Set binary as entrypoint
ENTRYPOINT /go/bin/nextbus-service /etc/nextbus-service/config.toml

# Expose default port (8080)
EXPOSE 8080
