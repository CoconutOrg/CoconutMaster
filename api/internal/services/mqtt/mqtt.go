package mqtt

import (
	"log/slog"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func NewMqttClient(opts *MQTT.ClientOptions, topics map[string]byte) (*MqttClient, error) {
	opts.SetOnConnectHandler(MqttClientOnConnectHandler)
	opts.SetConnectionLostHandler(MqttClientLostConnectionHandler)
	opts.SetDefaultPublishHandler(MqttClientMessagePubHandler)

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	if token := client.SubscribeMultiple(topics, MqttClientMessagePubHandler); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	
	return &MqttClient{ client: client }, nil
}

func MqttClientOnConnectHandler(client MQTT.Client) {
	slog.Info("MQTT: Connected!")
}

func MqttClientLostConnectionHandler(client MQTT.Client, err error) {
	slog.Error("MQTT: Lost connection!", "error", err.Error())
}

func MqttClientMessagePubHandler(client MQTT.Client, msg MQTT.Message) {
	slog.Info("MQTT: Recieved message!", "topic", msg.Topic(), "message", msg.Payload())
}

type IMqttClient interface {
	Disconnect()
	Publish(topic string, qos byte, retained bool, message string) error
}

type MqttClient struct {
	client MQTT.Client
}

func (c *MqttClient) Disconnect() {
	c.client.Disconnect(250)
	slog.Warn("MQTT: Disconnecting!")
}

func (c *MqttClient) Publish(topic string, qos byte, retained bool, message string) error {
	if token := c.client.Publish(topic, qos, retained, message); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	
	return nil
}