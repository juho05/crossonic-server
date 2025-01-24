package connect

import (
	"context"
	"net"
	"sync"

	"github.com/gobwas/ws/wsutil"
	crossonic "github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/sonos"
	"github.com/juho05/log"
)

type sonosListeners struct {
	ctx           context.Context
	cancel        context.CancelFunc
	listeners     map[string]struct{}
	listenersLock sync.RWMutex
}

type ConnectionManager struct {
	userConnections map[string]*connections
	connectionCount int
	lock            sync.RWMutex

	sonosListeners     map[string]*sonosListeners
	sonosListenersLock sync.RWMutex

	db    repos.DB
	sonos *sonos.SonosController
}

func NewConnectionManager(db repos.DB) *ConnectionManager {
	cm := &ConnectionManager{
		userConnections: make(map[string]*connections),
		db:              db,
		sonosListeners:  make(map[string]*sonosListeners),
	}
	cm.sonos = sonos.NewController(cm.onNewSonosDevice, cm.onSonosDeviceRemoved)
	return cm
}

type connections struct {
	connections map[string]*Connection
	lock        sync.RWMutex
}

func (c *ConnectionManager) Connect(user, name string, platform DevicePlatform, conn net.Conn) string {
	id := crossonic.GenID()
	connection := &Connection{
		User:     user,
		Platform: platform,
		ID:       id,
		Name:     name,
		conn:     conn,
		manager:  c,
	}
	go connection.handle()
	c.lock.Lock()
	userConns, ok := c.userConnections[user]
	if !ok {
		userConns = &connections{
			connections: map[string]*Connection{
				id: connection,
			},
		}
		c.userConnections[user] = userConns
	} else {
		userConns.lock.Lock()
		userConns.connections[id] = connection
		userConns.lock.Unlock()
	}
	var startSonosTimer bool
	c.connectionCount++
	if c.connectionCount == 1 {
		startSonosTimer = true
	}
	c.lock.Unlock()
	log.Tracef("connect: new device for user %s: %s (%s)", user, name, id)
	userConns.lock.RLock()
	for _, cn := range userConns.connections {
		if cn.ID == id {
			continue
		}
		msg, err := newNewDeviceNotification(cn.Name, cn.ID, cn.Platform)
		if err != nil {
			log.Errorf("connect: send new device notification: %s", err)
		} else {
			msg.Target = id
			c.sendMessage(user, msg, "")
		}
	}
	userConns.lock.RUnlock()
	for _, d := range c.sonos.Devices() {
		msg, err := newNewDeviceNotification(d, sonosNameToID(d), DevicePlatformSpeaker)
		if err != nil {
			log.Errorf("connect: send new sonos device notification: %s", err)
		} else {
			msg.Target = id
			c.sendMessage(user, msg, "")
		}
	}
	if startSonosTimer {
		c.sonos.StartRefreshDevicesTimer()
	}
	msg, err := newNewDeviceNotification(name, id, platform)
	if err != nil {
		log.Errorf("connect: send new device notification: %s", err)
	} else {
		c.sendMessage(user, msg, id)
	}
	return id
}

func (c *ConnectionManager) Disconnect(user, id string) {
	c.lock.RLock()
	conns, ok := c.userConnections[user]
	c.lock.RUnlock()
	if ok {
		conns.lock.Lock()
		cn, ok := conns.connections[id]
		if ok {
			cn.Close()
			delete(conns.connections, id)
			log.Tracef("connect: device of user %s disconnected: %s (%s)", user, cn.Name, id)
			c.connectionCount--
			if c.connectionCount < 0 {
				log.Errorf("connect: connection count is negative: %d", c.connectionCount)
				c.connectionCount = 0
			}
			if c.connectionCount == 0 {
				c.sonos.StopRefreshDevicesTimer()
			}
			conns.lock.Unlock()

			if cn.listenID != nil && isSonosID(*cn.listenID) {
				name := sonosNameFromID(*cn.listenID)
				c.sonosListenersLock.Lock()
				l, ok := c.sonosListeners[name]
				if ok {
					l.listenersLock.Lock()
					delete(l.listeners, id)
					if len(l.listeners) == 0 {
						l.cancel()
						delete(c.sonosListeners, name)
					}
					l.listenersLock.Unlock()
				}
				c.sonosListenersLock.Unlock()
			}

			msg, err := newDeviceDisconnectedNotification(id)
			if err != nil {
				log.Errorf("connect: send device disconnected notification: %s", err)
			} else {
				c.sendMessage(user, msg, id)
			}
		} else {
			conns.lock.Unlock()
		}
	}
}

func (c *ConnectionManager) broadcastMessageToAllUsers(msg message) {
	data, err := msg.encode()
	if err != nil {
		log.Errorf("connect: broadcast message to all users: %s", err)
		return
	}
	c.lock.RLock()
	for _, userConn := range c.userConnections {
		userConn.lock.RLock()
		for _, c := range userConn.connections {
			err = wsutil.WriteServerText(c.conn, data)
			if err != nil {
				log.Errorf("connect: broadcast message to all users: write text to %s (%s): %s", c.Name, c.ID, err)
			}
		}
		userConn.lock.RUnlock()
	}
	c.lock.RUnlock()
}

func (c *ConnectionManager) sendMessageAllUsers(msg message, excludeID string) {
	c.lock.RLock()
	users := make([]string, 0, len(c.userConnections))
	for u := range c.userConnections {
		users = append(users, u)
	}
	c.lock.RUnlock()
	for _, u := range users {
		c.sendMessage(u, msg, excludeID)
	}
}

func (c *ConnectionManager) sendMessage(user string, msg message, excludeID string) {
	conns := make(map[string]*Connection)
	if msg.Target == targetAll {
		c.lock.RLock()
		userConns, ok := c.userConnections[user]
		c.lock.RUnlock()
		if !ok {
			log.Errorf("connect: tried to send message to non-existent user '%s'", user)
			return
		}
		userConns.lock.RLock()
		conns = userConns.connections
		defer userConns.lock.RUnlock()
	} else {
		c.lock.RLock()
		userConns, ok := c.userConnections[user]
		c.lock.RUnlock()
		if ok {
			userConns.lock.RLock()
			cn, ok := userConns.connections[msg.Target]
			userConns.lock.RUnlock()
			if !ok {
				log.Errorf("connect: tried to send message to non-existent target '%s'", msg.Target)
				return
			}
			conns[cn.ID] = cn
		} else {
			log.Errorf("connect: tried to send message to non-existent target '%s'", msg.Target)
			return
		}
	}
	data, err := msg.encode()
	if err != nil {
		log.Errorf("connect: %s", err)
		return
	}
	for _, c := range conns {
		if c.ID == excludeID {
			continue
		}
		if msg.Target == c.ID || msg.Source == sourceServer || (c.listenID != nil && msg.Source == *c.listenID) {
			err = wsutil.WriteServerText(c.conn, data)
			if err != nil {
				log.Errorf("connect: write text to %s (%s): %s", c.Name, c.ID, err)
			}
		}
	}
}

func (c *ConnectionManager) Close() error {
	c.sonosListenersLock.Lock()
	for _, l := range c.sonosListeners {
		l.cancel()
	}
	c.sonosListenersLock.Unlock()
	c.lock.Lock()
	for _, conns := range c.userConnections {
		conns.lock.Lock()
		for _, c := range conns.connections {
			c.Close()
		}
		conns.lock.Unlock()
	}
	c.userConnections = nil
	c.lock.Unlock()
	return nil
}
