package connect

import (
	"net"
	"sync"

	"github.com/gobwas/ws/wsutil"
	"github.com/juho05/crossonic-server"
	"github.com/juho05/log"
)

type ConnectionManager struct {
	userConnections map[string]*connections
	lock            sync.RWMutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		userConnections: make(map[string]*connections),
	}
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
			conns.lock.Unlock()

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
