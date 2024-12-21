package server

import (
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func createTopicWithRetry(logger *logrus.Logger, brokerAddress, topic string, numPartitions, replicationFactor int) error {
	var createTopicErr error
	for i := 0; i < 10; i++ {
		createTopicErr = createTopic(webServer.logger, brokerAddress, topic, numPartitions, replicationFactor)
		if createTopicErr == nil {
			return nil
		}
		logger.Warnf("fail to create kafka topic: %v. retrying...", createTopicErr)
		time.Sleep(5 * time.Millisecond)
	}
	return createTopicErr
}

func createTopic(logger *logrus.Logger, brokerAddress, topic string, numPartitions, replicationFactor int) error {
	conn, err := kafka.Dial("tcp", brokerAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to Kafka broker: %w", err)
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions()
	if err != nil {
		return fmt.Errorf("failed to read partitions: %w", err)
	}

	for _, p := range partitions {
		if p.Topic == topic {
			logger.Infof("Topic %q already exists", topic)
			return nil
		}
	}

	topicConfig := kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     numPartitions,
		ReplicationFactor: replicationFactor,
	}

	err = conn.CreateTopics(topicConfig)
	if err != nil {
		return fmt.Errorf("failed to create topic: %w", err)
	}

	logger.Infof("Topic %q created successfully", topic)
	return nil
}
