package connect

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/log"
)

func (c *ConnectionManager) handleSonosMessage(user string, msg message) {
	if msg.Type != msgTypeCommand {
		return
	}
	device := sonosNameFromID(msg.Target)
	var err error
	switch msg.Op {
	case msgOpPlay:
		err = c.sonos.Play(device)
	case msgOpPause:
		err = c.sonos.Pause(device)
	case msgOpStop:
		err = c.sonos.Stop(device)
	case msgOpSpeakerSetCurrent:
		err = c.handleSetCurrent(user, device, msg)
	case msgOpSpeakerSetNext:
		err = c.handleSetNext(user, device, msg)
	}
	if err != nil {
		log.Errorf("handle message: %s: %s", msg.Op, err)
		return
	}
}

func (c *ConnectionManager) handleSetCurrent(user, device string, msg message) error {
	var payload speakerSetCurrentPayload
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		return fmt.Errorf("handle set current: decode: %w", err)
	}
	u, err := c.store.FindUser(context.Background(), user)
	if err != nil {
		return fmt.Errorf("handle set current: find user: %w", err)
	}
	password, err := sqlc.DecryptPassword(u.EncryptedPassword)
	if err != nil {
		return fmt.Errorf("handle set current: decrypt password: %w", err)
	}
	hash, salt := generateAuth(password)
	currentURL := fmt.Sprintf("%s/rest/stream.mp3?u=%s&t=%s&s=%s&c=sonos&v=1.16.1&id=%s&format=mp3&maxBitRate=320&timeOffset=%d", config.BaseURL(), user, hash, salt, payload.SongID, payload.TimeOffset)
	var nextURL *string
	if payload.NextID != nil {
		url := fmt.Sprintf("%s/rest/stream.mp3?u=%s&t=%s&s=%s&c=sonos&v=1.16.1&id=%s&format=mp3&maxBitRate=320", config.BaseURL(), user, hash, salt, *payload.NextID)
		nextURL = &url
	}
	err = c.sonos.SetCurrent(device, currentURL, nextURL)
	if err != nil {
		return fmt.Errorf("handle set current: %w", err)
	}
	return nil
}

func (c *ConnectionManager) handleSetNext(user, device string, msg message) error {
	var payload speakerSetNextPayload
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		return fmt.Errorf("handle set next: decode: %w", err)
	}
	u, err := c.store.FindUser(context.Background(), user)
	if err != nil {
		return fmt.Errorf("handle set next: find user: %w", err)
	}
	password, err := sqlc.DecryptPassword(u.EncryptedPassword)
	if err != nil {
		return fmt.Errorf("handle set next: decrypt password: %w", err)
	}
	hash, salt := generateAuth(password)
	var nextURL *string
	if payload.SongID != nil {
		url := fmt.Sprintf("%s/rest/stream.mp3?u=%s&t=%s&s=%s&c=sonos&v=1.16.1&id=%s&format=mp3&maxBitRate=320", config.BaseURL(), user, hash, salt, *payload.SongID)
		nextURL = &url
	}
	err = c.sonos.SetNext(device, nextURL)
	if err != nil {
		return fmt.Errorf("handle set next: %w", err)
	}
	return nil
}

func (c *ConnectionManager) onNewSonosDevice(name string) {
	msg, err := newNewDeviceNotification(name, sonosNameToID(name), DevicePlatformSpeaker)
	if err != nil {
		log.Errorf("new sonos device: %w", err)
		return
	}
	c.broadcastMessageToAllUsers(msg)
}

func (c *ConnectionManager) onSonosDeviceRemoved(name string) {
	msg, err := newDeviceDisconnectedNotification(sonosNameToID(name))
	if err != nil {
		log.Errorf("sonos device removed: %w", err)
		return
	}
	c.broadcastMessageToAllUsers(msg)
}

func sonosNameToID(name string) string {
	return fmt.Sprintf("server_sonos_%s", name)
}

func sonosNameFromID(id string) string {
	return strings.TrimPrefix(id, "server_sonos_")
}

func isSonosID(id string) bool {
	return strings.HasPrefix(id, "server_sonos_")
}

func generateAuth(password string) (hash string, salt string) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	saltBytes := make([]byte, 12)
	for i := range saltBytes {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			panic(err)
		}
		saltBytes[i] = letters[num.Int64()]
	}
	salt = string(saltBytes)

	hashBytes := md5.Sum([]byte(password + salt))
	hash = hex.EncodeToString(hashBytes[:])
	return hash, salt
}
