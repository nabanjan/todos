package main

import (
	"fmt"
	"sync"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

func kafkaConsumer(wg *sync.WaitGroup) {
	defer wg.Done()
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":     "localhost",
		"group.id":              "myGroup",
		"auto.offset.reset":     "earliest",
		"broker.address.family": "v4",
	})

	if err != nil {
		panic(err)
	}
	defer c.Close()

	c.SubscribeTopics([]string{"myTopic", "^aRegex.*[Tt]opic"}, nil)

	for {
		msg, err := c.ReadMessage(-1)
		if err == nil {
			fmt.Printf("Message on %s: %s\n", msg.TopicPartition, string(msg.Value))
		} else {
			// The client will automatically try to recover from all errors.
			fmt.Printf("Consumer error: %v (%v)\n", err, msg)
		}
	}

}
