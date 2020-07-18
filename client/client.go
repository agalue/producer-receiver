// @author Alejandro Galue <agalue@opennms.org>

// Package client implements a kafka consumer that works for OpenNMS Kafka Producer messages.
package client

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/agalue/producer-receiver/protobuf/producer"
	"github.com/golang/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

const (
	// EventKind Event Message Kind
	EventKind = "event"
	// AlarmKind Alarm Message Kind
	AlarmKind = "alarm"
	// NodeKind Node Message Kind
	NodeKind = "node"
	// EdgeKind Topology Edge Message Kind
	EdgeKind = "edge"
	// MetricKind Metric Message Kind
	MetricKind = "metric"
)

// ValidKinds list of valid message kinds.
var ValidKinds = []string{EventKind, AlarmKind, NodeKind, EdgeKind, MetricKind}

// ProcessMessage defines the action to execute after successfully received a message.
type ProcessMessage func(msg []byte)

// KafkaClient represents a simple Kafka consumer cli.
type KafkaClient struct {
	Bootstrap   string // The Kafka Server Bootstrap string.
	Topic       string // The name of the Kafka Topic.
	GroupID     string // The name of the Consumer Group ID.
	MessageKind string // Kind of the message to process: event, alarm, node, metric
	Parameters  string // CSV of KVP for the Kafka Consumer Parameters.

	consumer *kafka.Consumer
	stopping bool

	msgProcessed prometheus.Counter
}

// Creates the Kafka Configuration Map.
func (cli *KafkaClient) createConfig() *kafka.ConfigMap {
	config := &kafka.ConfigMap{
		"bootstrap.servers":     cli.Bootstrap,
		"group.id":              cli.GroupID,
		"session.timeout.ms":    6000,
		"broker.address.family": "v4",
	}
	if cli.Parameters != "" {
		for _, kv := range strings.Split(cli.Parameters, ",") {
			array := strings.Split(kv, "=")
			if len(array) == 2 {
				if err := config.SetKey(array[0], array[1]); err != nil {
					log.Printf("invalid property %s=%s: %v", array[0], array[1], err)
				}
			} else {
				log.Printf("invalid key-value pair %s", kv)
			}
		}
	}
	return config
}

func (cli *KafkaClient) validate() error {
	if cli.Topic == "" {
		return fmt.Errorf("source topic cannot be empty")
	}
	if cli.MessageKind == "" {
		return fmt.Errorf("message kind cannot be empty")
	}
	set := make(map[string]bool, len(ValidKinds))
	for _, s := range ValidKinds {
		set[s] = true
	}
	if _, ok := set[cli.MessageKind]; !ok {
		return fmt.Errorf("invalid message kind %s. Valid options: %s", cli.MessageKind, strings.Join(ValidKinds, ", "))
	}
	return nil
}

func (cli *KafkaClient) process(msg *kafka.Message, action ProcessMessage) {
	var data proto.Message
	switch cli.MessageKind {
	case EventKind:
		data = &producer.Event{}
	case AlarmKind:
		data = &producer.Alarm{}
	case NodeKind:
		data = &producer.Node{}
	case EdgeKind:
		data = &producer.TopologyEdge{}
	case MetricKind:
		data = &producer.CollectionSet{}
	}
	if err := proto.Unmarshal(msg.Value, data); err != nil {
		log.Printf("invalid %s message received: %v", cli.MessageKind, err)
	}
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err == nil {
		cli.msgProcessed.Inc()
		log.Printf("message received with key '%v' on partition %d (offset %d)", msg.Key, msg.TopicPartition.Partition, msg.TopicPartition.Offset)
		action(jsonBytes)
	} else {
		log.Printf("cannot convert GPB to JSON: %v", err)
	}
	_, err = cli.consumer.CommitMessage(msg) // If there are errors on the action, the message won't be reprocessed.
	if err != nil {
		log.Printf("error committing message: %v", err)
	}
}

func (cli *KafkaClient) byteCount(b float64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%f B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", b/float64(div), "KMGTPE"[exp])
}

func (cli *KafkaClient) showStats(sts *kafka.Stats) {
	// https://github.com/edenhill/librdkafka/blob/master/STATISTICS.md
	var stats map[string]interface{}
	json.Unmarshal([]byte(sts.String()), &stats)
	log.Printf("statistics: %v messages (%v) consumed", stats["rxmsgs"], cli.byteCount(stats["rxmsg_bytes"].(float64)))
}

// Initialize builds the Kafka consumer object and the cache for chunk handling.
func (cli *KafkaClient) Initialize() error {
	var err error
	if err = cli.validate(); err != nil {
		return err
	}
	cli.msgProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "onms_producer_processed_messages_total",
		Help: "The total number of processed messages",
	})
	log.Printf("creating consumer for topic %s at %s", cli.Topic, cli.Bootstrap)
	cli.consumer, err = kafka.NewConsumer(cli.createConfig())
	if err != nil {
		return fmt.Errorf("could not create consumer: %v", err)
	}
	err = cli.consumer.Subscribe(cli.Topic, nil)
	if err != nil {
		return fmt.Errorf("cannot subscribe to topic %s: %v", cli.Topic, err)
	}
	return nil
}

// Start registers the consumer for the chosen topic, and reads messages from it on an infinite loop.
// It is recommended to use it within a Go Routine as it is a blocking operation.
func (cli *KafkaClient) Start(action ProcessMessage) {
	jsonBytes, _ := json.MarshalIndent(cli, "", "  ")
	log.Printf("starting kafka consumer: %s", string(jsonBytes))

	cli.stopping = false
	for {
		if cli.stopping {
			return
		}
		event := cli.consumer.Poll(500)
		switch e := event.(type) {
		case *kafka.Message:
			cli.process(e, action)
		case kafka.Error:
			log.Printf("consumer error %v", e)
		case *kafka.Stats:
			cli.showStats(e)
		}
	}
}

// Stop terminates the Kafka consumer and waits for the execution of all pending action handlers.
func (cli *KafkaClient) Stop() {
	log.Println("stopping consumer")
	cli.stopping = true
	cli.consumer.Close()
	log.Println("good bye!")
}

// Bootstrap function
