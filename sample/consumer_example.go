// go get -u "github.com/apache/pulsar-client-go/pulsar"

package sample

import (
	"fmt"
	"github.com/apache/pulsar-client-go/pulsar"
	"log"
)

func main() {
	fmt.Println("consumer")

	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:            "pulsar://$IP:$PORT",
		Authentication: pulsar.NewAuthenticationToken("$TOKEN"),
	})
	if err != nil {
		fmt.Println(err)
		panic(fmt.Errorf("could not instantiate Pulsar client: %v", err))
	}
	defer client.Close()

	var consumer pulsar.Consumer
	var msg = make(chan pulsar.ConsumerMessage, 100)
	consumer, err = client.Subscribe(pulsar.ConsumerOptions{
		Topic:            "apache/pulsar/test-topic",
		SubscriptionName: "sub",
		Type:             pulsar.Exclusive,
		MessageChannel:   msg,
	})

	if err != nil {
		panic(fmt.Errorf("could not subscribe from Pulsar: %v", err))
	}
	defer consumer.Close()

	for msg := range msg {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Received message msgId: %#v -- content: '%s'\n", msg.ID(), string(msg.Payload()))
		consumer.Ack(msg)
	}
	if err := consumer.Unsubscribe(); err != nil {
		log.Fatal(err)
	}
}
