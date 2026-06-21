package repo

import (
	"log/slog"

	"github.com/CoconutOrg/CoconutMaster/internal/json"
	"github.com/CoconutOrg/CoconutMaster/internal/services/mqtt"
)

func (mr *MqttRepository) Close() {
	mr.client.Disconnect()
}

func NewMqttRepo(client *mqtt.MqttClient) (*MqttRepository) {
	result := &MqttRepository{
		client: client,
	}

	return result
}

type IMqttRepository interface {
	Close()
	PublishRegiserConfirmUserMessage(email string, code string) error
}

type MqttRepository struct {
	client *mqtt.MqttClient
}

func (mr *MqttRepository) PublishRegiserConfirmUserMessage(email string, code string) error {
	msg := &RegiserConfirmUserMessageParams{
		Subtopic: "Register",
		Email: email,
		Code: code,
	}

	msgJson, err := json.ToJsonString(msg)
	if err != nil {
		return err
	}
	slog.Info(msgJson)

	err = mr.client.Publish("user", byte(1), false, msgJson)

	return err
}

type RegiserConfirmUserMessageParams struct {
	Subtopic string `json:"subtopic"`
	Email string `json:"email"`
	Code string `json:"code"`
}