package connect

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strings"

	"github.com/gobwas/ws/wsutil"
	"github.com/juho05/log"
)

type DevicePlatform string

const (
	DevicePlatformPhone   DevicePlatform = "phone"
	DevicePlatformWeb     DevicePlatform = "web"
	DevicePlatformDesktop DevicePlatform = "desktop"
	DevicePlatformSpeaker DevicePlatform = "speaker"
	DevicePlatformUnknown DevicePlatform = "unknown"
)

func (d DevicePlatform) Valid() bool {
	return d == DevicePlatformPhone || d == DevicePlatformWeb || d == DevicePlatformDesktop || d == DevicePlatformSpeaker || d == DevicePlatformUnknown
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
		msg.Source = c.ID
		if msg.Target == "server" || strings.HasPrefix(msg.Target, "server_") {
			c.handleMessage(msg)
			continue
		}
		c.manager.sendMessage(c.User, msg, c.ID)
	}
}

func (c *Connection) handleMessage(msg message) {
	if msg.Op == msgOpListen {
		c.handleListenMessage(msg)
		return
	}
	if isSonosID(msg.Target) {
		c.manager.handleSonosMessage(c.User, msg)
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

	if c.listenID != nil && isSonosID(*c.listenID) {
		name := sonosNameFromID(*c.listenID)
		c.manager.sonosListenersLock.Lock()
		l, ok := c.manager.sonosListeners[name]
		if ok {
			l.listenersLock.Lock()
			delete(l.listeners, c.ID)
			if len(l.listeners) == 0 {
				l.cancel()
				delete(c.manager.sonosListeners, name)
			}
			l.listenersLock.Unlock()
		}
		c.manager.sonosListenersLock.Unlock()
	}

	if payload.ID != nil {
		log.Tracef("connect: %s (%s) of %s started listening to %s", c.Name, c.ID, c.User, *payload.ID)
	} else {
		log.Tracef("connect: %s (%s) of %s stopped listening to anything", c.Name, c.ID, c.User)
	}
	c.listenID = payload.ID

	if payload.ID != nil {
		if isSonosID(*payload.ID) {
			name := sonosNameFromID(*payload.ID)
			c.manager.sonosListenersLock.Lock()
			l, ok := c.manager.sonosListeners[name]
			if !ok {
				ctx, cancel := context.WithCancel(context.Background())
				l = &sonosListeners{
					ctx:       ctx,
					cancel:    cancel,
					listeners: make(map[string]struct{}),
				}
				err = c.manager.sonos.OnEvent(ctx, name, func(sonosState string) {
					pos, err := c.manager.sonos.GetPosition(name)
					if err != nil {
						log.Errorf("sonos: on event: %s", err)
						pos = 0
					}
					var state speakerStatus
					switch sonosState {
					case "STOPPED", "NO_MEDIA_PRESENT":
						state = speakerStatusStopped
					case "PAUSED_PLAYBACK":
						state = speakerStatusPaused
					case "PLAYING":
						state = speakerStatusPlaying
					case "TRANSITIONING":
						state = speakerStatusLoading
					case "advance":
						state = speakerStatusAdvance
					default:
						log.Errorf("sonos: on event: unknown sonos state: %s", sonosState)
						state = speakerStatusLoading
					}
					eventMsg, err := newSpeakerStateNotification(sonosNameToID(name), state, pos)
					if err != nil {
						log.Errorf("sonos: on event: %s", err)
						return
					}
					c.manager.sendMessageAllUsers(eventMsg, "")
				})
				if err != nil {
					cancel()
					log.Errorf("sonos: on event: %s", err)
					return
				}
				c.manager.sonosListeners[name] = l
			}
			l.listenersLock.Lock()
			l.listeners[c.ID] = struct{}{}
			l.listenersLock.Unlock()
			c.manager.sonosListenersLock.Unlock()
		} else {
			m, err := newUpdateListenerRequest(*payload.ID, c.ID)
			if err != nil {
				log.Errorf("connect: request listener update: %w", err)
				return
			}
			c.manager.sendMessage(c.User, m, c.ID)
		}
	}
}

func (c *Connection) Close() error {
	return c.conn.Close()
}
