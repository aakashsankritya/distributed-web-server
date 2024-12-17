package main

import (
	"coding/server"
)

func main() {
	server := server.NewWebServer("webserver.log", "localhost:6379")
	server.Start(":8080")
}
