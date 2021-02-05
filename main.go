// Simple consumer/producer
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/agalue/producer-receiver/client"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var showMsg bool
	cli := client.KafkaClient{}
	flag.StringVar(&cli.Bootstrap, "bootstrap", "localhost:9092", "kafka bootstrap server")
	flag.StringVar(&cli.Topic, "topic", "alarms", "kafka source topic with OpenNMS Producer GPB messages")
	flag.StringVar(&cli.GroupID, "group-id", "producer-receiver", "kafka consumer group ID")
	flag.StringVar(&cli.MessageKind, "message-kind", client.AlarmKind, "source topic message kind; valid options: "+strings.Join(client.ValidKinds, ", "))
	flag.StringVar(&cli.Parameters, "parameters", "", "optional kafka consumer parameters as a CSV of Key-Value pairs")
	flag.BoolVar(&showMsg, "show-msg", true, "show message content in JSON")
	flag.Parse()

	log.Println("starting consumer")
	if err := cli.Initialize(); err != nil {
		panic(err)
	}
	log.Println("consumer started")

	go cli.Start(func(key []byte, msg []byte) {
		/////////////////////////////////////////////
		// TODO Implement your custom actions here //
		/////////////////////////////////////////////
		if showMsg {
			log.Printf("message received with (key: '%s'): %s", string(msg), string(msg))
		}
	})

	go func() {
		port := 8181
		log.Printf("Starting Prometheus Metrics Server on port %d", port)
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	cli.Stop()
}
