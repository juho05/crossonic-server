package connect

import (
	"encoding/json"
	"fmt"
)

type messageOp string

const (
	msgOpNewDevice          messageOp = "new-device"
	msgOpDeviceDisconnected messageOp = "device-disconnected"
	msgOpUpdateListener     messageOp = "update-listener"

	msgOpListen messageOp = "listen"
)

type messageType string

const (
	msgTypeNotification messageType = "notification"
	msgTypeCommand      messageType = "command"
	msgTypeRequest      messageType = "request"
)

const sourceServer = "server"
const targetAll = "all"

type message struct {
	Op      messageOp       `json:"op"`
	Type    messageType     `json:"type"`
	Source  string          `json:"source,omitempty"`
	Target  string          `json:"target,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type newDevicePayload struct {
	Name     string         `json:"name"`
	ID       string         `json:"id"`
	Platform DevicePlatform `json:"platform"`
}

type deviceDisconnectedPayload struct {
	ID string `json:"id"`
}

type listenPayload struct {
	ID *string `json:"id"`
}

type updateListenerPayload struct {
	ID string `json:"id"`
}

func newNewDeviceNotification(name, id string, platform DevicePlatform) (message, error) {
	data, err := json.Marshal(newDevicePayload{
		Name:     name,
		ID:       id,
		Platform: platform,
	})
	if err != nil {
		return message{}, fmt.Errorf("new device notification: %w", err)
	}
	return message{
		Op:      msgOpNewDevice,
		Type:    msgTypeNotification,
		Source:  sourceServer,
		Target:  targetAll,
		Payload: data,
	}, nil
}

func newDeviceDisconnectedNotification(id string) (message, error) {
	data, err := json.Marshal(deviceDisconnectedPayload{
		ID: id,
	})
	if err != nil {
		return message{}, fmt.Errorf("device disconnected notification: %w", err)
	}
	return message{
		Op:      msgOpDeviceDisconnected,
		Type:    msgTypeNotification,
		Source:  sourceServer,
		Target:  targetAll,
		Payload: data,
	}, nil
}

func newUpdateListenerRequest(id, listenerID string) (message, error) {
	data, err := json.Marshal(updateListenerPayload{
		ID: listenerID,
	})
	if err != nil {
		return message{}, fmt.Errorf("update listener request: %w", err)
	}
	return message{
		Op:      msgOpUpdateListener,
		Type:    msgTypeRequest,
		Source:  sourceServer,
		Target:  id,
		Payload: data,
	}, nil
}

func (m *message) encode() ([]byte, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("encode message: %w", err)
	}
	return data, nil
}
