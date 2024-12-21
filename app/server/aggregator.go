package server

import (
	"bytes"
	"encoding/json"
	"io"

	"net/http"

	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

type Event struct {
	UniqueRequestCount int64     `json:"unique_request_count"`
	EventTime          time.Time `json:"event_time"`
}

func (s *WebServer) StartAggregator() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			err := s.aggregateAndCleanRedis()
			if err != nil {
				s.logger.Errorf("Error during aggregation: %v", err)
			}
		}
	}
}

func (s *WebServer) acquireGlobalLock() bool {
	acquired, err := s.redisClient.SetNX(s.config.AGGREGATOR_GLOBAL_LOCK, true, 1*time.Minute).Result()
	if err != nil {
		s.logger.Errorf("error while acquiring global redis lock %v", err)
		return false
	}
	return acquired
}

func (s *WebServer) aggregateAndCleanRedis() error {
	acquired := s.acquireGlobalLock()
	if !acquired {
		s.logger.Info("some other node processing aggregation")
		return nil
	}

	uniqueReqCount, err := s.redisClient.SCard(s.config.REQUEST_SET_KEY).Result()
	if err != nil {
		return fmt.Errorf("failed to fetch keys from redis: %w", err)
	}
	s.publishEventToKafka(uniqueReqCount, time.Now())
	// flush cached requestIds from redis
	_, err = s.redisClient.Del(s.config.REQUEST_SET_KEY).Result()
	if err != nil {
		return fmt.Errorf("error flushing keys from Redis: %w", err)
	}
	s.logger.Infof("Total unique requests in last 1 min: %d", uniqueReqCount)
	return nil
}

func (s *WebServer) callExternalAPI(url, method string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		s.logger.Errorf("Error creating request: %v", err)
		return nil, err
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		s.logger.Errorf("Error calling endpoint API: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Errorf("Error reading response body: %v", err)
		return nil, err
	}
	s.logger.Infof("API Response Status: %s", resp.Status)
	s.logger.Infof("API Response Body: %s", string(responseBody))
	return responseBody, nil
}

func (s *WebServer) publishEventToKafka(uniqueReqCount int64, eventTime time.Time) {
	event := &Event{UniqueRequestCount: uniqueReqCount, EventTime: eventTime}

	message, err := json.Marshal(event)
	if err != nil {
		s.logger.Errorf("failed to marshal event: %v", err)
		return
	}
	err = s.kafkaProducer.WriteMessages(s.ctx, kafka.Message{
		Key:   []byte{},
		Value: []byte(message),
	})
	if err != nil {
		s.logger.Errorf("failed to publish event to kafka: %v", err)
	}
}
