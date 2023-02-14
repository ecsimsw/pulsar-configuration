// go get -u "github.com/apache/pulsar-client-go/pulsar"

package sample

import (
	"context"
	"fmt"
	"github.com/apache/pulsar-client-go/pulsar"
	"log"
)

func main() {
	fmt.Println("producer")

	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:            "pulsar://$IP:$PORT",
		Authentication: pulsar.NewAuthenticationToken("$TOKEN"),
	})

	producer, err := client.CreateProducer(pulsar.ProducerOptions{
		Topic: "apache/pulsar/test-topic",
	})

	if err != nil {
		log.Fatal(err)
	}

	_, err = producer.Send(context.Background(), &pulsar.ProducerMessage{
		Payload: []byte("hi"),
	})
	defer producer.Close()

	if err != nil {
		fmt.Println("Failed to publish message", err)
	}
	fmt.Println("Published message")
}
