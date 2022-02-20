package mqtt

import (
	"log"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofrs/uuid"
)

const (
	pushTopic = "repo/push"
)

type fn func(string)

type MqttClient struct {
	client mqtt.Client
	logger log.Logger
}

func NewMQTTClient(addr string, logger log.Logger) MqttClient {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(addr)
	var clientID string
	u, _ := uuid.NewV4()
	clientID = u.String()
	opts.SetClientID(clientID)
	opts.OnConnect = func(client mqtt.Client) {
		logger.Println("Connected to MQTT server")
	}
	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		logger.Fatalf("Connection to MQTT server lost: %v", err)
	}
	client := mqtt.NewClient(opts)

	return MqttClient{
		client,
		logger,
	}
}

func (c MqttClient) Connect() {
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		// todo: return error instead of panic
		panic(token.Error())
	}
}

func (c MqttClient) Cleanup() {
	c.client.Disconnect(250)
}

func (c MqttClient) Subscribe(topic string, subscribeHandler fn) error {
	var wg sync.WaitGroup
	wg.Add(1)

	if token := c.client.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		subscribeHandler(string(msg.Payload()))
	}); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (c MqttClient) Publish(topic, message string) error {
	token := c.client.Publish(topic, 0, false, message)
	token.Wait()
	return token.Error()
}
