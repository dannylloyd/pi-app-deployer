package mqtt

import (
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofrs/uuid"
)

type fn func(string)

type MqttClient struct {
	client mqtt.Client
}

// func NewMQTTClient(addr string, connectHandler func(client mqtt.Client), connectionLostHandler func(client mqtt.Client, err error)) MqttClient {
func NewMQTTClient(addr string, connectHandler func(client mqtt.Client), connectionLostHandler func(client mqtt.Client, err error)) MqttClient {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(addr)
	var clientID string
	u, _ := uuid.NewV4()
	clientID = u.String()
	opts.SetClientID(clientID)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectionLostHandler
	client := mqtt.NewClient(opts)

	return MqttClient{
		client,
	}
}

func (c MqttClient) Connect() error {
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
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
	// TODO: look into QoS
	token := c.client.Publish(topic, 0, false, message)
	token.Wait()
	return token.Error()
}
