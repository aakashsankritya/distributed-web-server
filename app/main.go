package main

import (
	"coding/server"
	"fmt"
	"os"
)

func main() {
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	appName := os.Getenv("APPNAME")
	fmt.Printf("Redis config is: Host: %s and port: %s \n", redisHost, redisPort)
	server := server.NewWebServer(appName, fmt.Sprintf("%s:%s", redisHost, redisPort))
	server.Start(":8080")
}
