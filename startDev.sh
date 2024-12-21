#!/bin/bash

docker-compose down

cd app
# build binary
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o ./out/webserver .

cd ..
# clean logs
rm -rf logs/*

# build docker image
docker build -t webserver .

docker-compose up -d 
