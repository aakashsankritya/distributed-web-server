package main

import (
	"coding/server"
	"fmt"
	"os"
)

func main() {
	serverPort := os.Getenv("SERVER_PORT")
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	kafkaUrl := os.Getenv("KAFKA_BROKER_ADDR")
	appName := os.Getenv("APPNAME")
	configFilePath := os.Getenv("CONFIG_FILE")
	fmt.Printf("Redis config is: Host: %s and port: %s \n", redisHost, redisPort)
	server := server.NewWebServer(appName, fmt.Sprintf("%s:%s", redisHost, redisPort), kafkaUrl, configFilePath)
	server.Start(fmt.Sprintf(":%s", serverPort))
}
