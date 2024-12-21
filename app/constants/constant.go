package constants

import (
	"encoding/json"
	"fmt"
	"os"
)

// const (
// 	REQ_ID                     = "REQ_ID:"
// 	AGGREGATED_LOG_FILE_NAME   = "aggregated-data"
// 	AGGREGATOR_GLOBAL_LOCK     = "AGGREGATOR_LOCK"
// 	WORKER_POOL_SIZE           = 100
// 	WORKER_CHANNEL_BUFFER_SIZE = 100000
// )

type Config struct {
	REQ_ID                     string `json:"REQ_ID"`
	AGGREGATED_LOG_FILE_NAME   string `json:"AGGREGATED_LOG_FILE_NAME"`
	AGGREGATOR_GLOBAL_LOCK     string `json:"AGGREGATOR_GLOBAL_LOCK"`
	WORKER_POOL_SIZE           int    `json:"WORKER_POOL_SIZE"`
	WORKER_CHANNEL_BUFFER_SIZE int    `json:"WORKER_CHANNEL_BUFFER_SIZE"`
	REQUEST_SET_KEY            string `json:"REQUEST_SET_KEY"`
	KAFKA_TOPIC                string `json:"KAFKA_TOPIC"`
}

func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("\n failed to load config from file: %s, error: %v", filePath, err)
		os.Exit(1)
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}
