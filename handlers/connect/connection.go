package connect

import (
	"encoding/json"
	"errors"
	"io"
	"net"

	"github.com/gobwas/ws/wsutil"
	"github.com/juho05/log"
)

type DevicePlatform string

const (
	DevicePlatformPhone   DevicePlatform = "phone"
	DevicePlatformWeb     DevicePlatform = "web"
	DevicePlatformDesktop DevicePlatform = "desktop"
	DevicePlatformUnknown DevicePlatform = "unknown"
)

func (d DevicePlatform) Valid() bool {
	return d == DevicePlatformPhone || d == DevicePlatformWeb || d == DevicePlatformDesktop || d == DevicePlatformUnknown
}

type Connection struct {
	User     string         `json:"-"`
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Platform DevicePlatform `json:"platform"`

	listenID *string `json:"-"`

	conn    net.Conn           `json:"-"`
	manager *ConnectionManager `json:"-"`
}

func (c *Connection) handle() {
	defer func() {
		c.manager.Disconnect(c.User, c.ID)
	}()

	for {
		msgData, err := wsutil.ReadClientText(c.conn)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Errorf("connect: read message from %s %s (%s): %s", c.User, c.Name, c.ID, err)
			}
			return
		}
		var msg message
		err = json.Unmarshal(msgData, &msg)
		if err != nil {
			log.Errorf("connect: decode message from %s %s (%s): %s", c.User, c.Name, c.ID, err)
			continue
		}
		if msg.Op == msgOpListen {
			c.handleListenMessage(msg)
		}
		c.manager.sendMessage(c.User, msg, c.ID)
	}
}

func (c *Connection) handleListenMessage(msg message) {
	if msg.Type != msgTypeCommand {
		return
	}
	var payload listenPayload
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		log.Errorf("connect: received invalid listen message payload")
		return
	}
	c.listenID = payload.ID

	if payload.ID != nil {
		m, err := newUpdateListenerRequest(*payload.ID, c.ID)
		if err != nil {
			log.Errorf("connect: request listener update: %w", err)
			return
		}
		c.manager.sendMessage(c.User, m, c.ID)
	}
}

func (c *Connection) Close() error {
	return c.conn.Close()
}
