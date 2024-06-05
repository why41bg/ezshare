package ws

import (
	"fmt"
	"github.com/ezshare/server/ws/outgoing"
	"github.com/gorilla/websocket"
	"github.com/rs/xid"
	"github.com/rs/zerolog/log"
	"net"
	"time"
)

const (
	WriteTimeout = time.Second * 2
)

type Client struct {
	conn *websocket.Conn
	info ClientInfo
	once once
	read chan<- ClientMessage
}

type ClientInfo struct {
	ID                xid.ID
	RoomID            string
	Authenticated     bool
	AuthenticatedUser string
	Write             chan outgoing.Message
	Close             chan string
	Addr              net.IP
}

type ClientMessage struct {
	Info     ClientInfo
	Incoming Event
}

func newClient(conn *websocket.Conn, read chan ClientMessage, authenticatedUser string, authenticated bool) *Client {
	// 创建一个新的Client对象包装WebSocket连接，并设置其关闭时的回调函数
	c := &Client{
		conn: conn,
		info: ClientInfo{
			ID:                xid.New(),
			RoomID:            "",
			Authenticated:     authenticated,
			AuthenticatedUser: authenticatedUser,
			Write:             make(chan outgoing.Message, 1),
			Close:             make(chan string, 1),
			Addr:              conn.RemoteAddr().(*net.TCPAddr).IP,
		},
		read: read,
	}
	log.Debug().Str("clientId", c.info.ID.String()).Str("user", c.info.AuthenticatedUser).Msg("New client created")
	conn.SetCloseHandler(func(code int, text string) error {
		message := websocket.FormatCloseMessage(code, text)
		log.Debug().Str("clientId", c.info.ID.String()).Str("reason", text).Int("code", code).Msg("WebSocket Close")
		return conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(WriteTimeout))
	})
	return c
}

// Close 主动关闭WebSocket连接，并且向read通道一个通知
func (c *Client) Close() {
	c.once.Do(func() {
		_ = c.conn.Close()
		log.Debug().Str("clientId", c.info.ID.String()).Str("user", c.info.AuthenticatedUser).Msg("WebSocket Close")
		go func() {
			c.read <- ClientMessage{
				Info:     c.info,
				Incoming: &Disconnected{},
			}
		}()
	})
}

// startReading try to get the next reader from the websocket connection.
// If the message type is websocket.BinaryMessage, close the connection. If
// the message is websocket.TextMessage, then parse it and send it to the Rooms.
func (c *Client) startReading(pongWait time.Duration) {
	defer c.Close()
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(appData string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		t, m, err := c.conn.NextReader()
		if err != nil {
			log.Error().Err(err).Str("clientId", c.info.ID.String()).Msg("Failed to get next reader")
			return
		}
		if t == websocket.BinaryMessage {
			_ = c.conn.CloseHandler()(websocket.CloseUnsupportedData, fmt.Sprintf("Unsupported binary message type: %s", err))
			return
		}
		event, err := ReadTypedIncoming(m)
		if err != nil {
			_ = c.conn.CloseHandler()(websocket.CloseNormalClosure, fmt.Sprintf("Malformed message: %s", err))
			return
		}
		log.Debug().Str("clientId", c.info.ID.String()).Str("user", c.info.AuthenticatedUser).Msg("WebSocket Receive")
		c.read <- ClientMessage{Info: c.info, Incoming: event}
	}
}

// startWriteHandler reads messages from the write channel and sends them to the
// websocket connection. It also sends ping to the connection at regular intervals.
// When it occurs, it will close the connection and stop the pingTicker right now.
func (c *Client) startWriteHandler(pingPeriod time.Duration) {
	pingTicker := time.NewTicker(pingPeriod)
	dead := false
	conClosed := func() {
		dead = true
		c.Close()
		pingTicker.Stop()
	}
	defer conClosed()
	defer func() {
		log.Debug().Str("clientId", c.info.ID.String()).Str("user", c.info.AuthenticatedUser).Msg("WebSocket Done")
	}()

	for {
		select {
		case reason := <-c.info.Close:
			if reason == CloseDone {
				return
			} else {
				_ = c.conn.CloseHandler()(websocket.CloseNormalClosure, reason)
				conClosed()
			}
		case message := <-c.info.Write:
			if dead {
				log.Debug().Str("clientId", c.info.ID.String()).Str("user", c.info.AuthenticatedUser).Msg("WebSocket write on dead connection")
				continue
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
			typed, err := ToTypedOutgoing(message)
			if err != nil {
				log.Error().Err(err).Msg("Could not get typed message, exiting conn")
				conClosed()
				continue
			}
			if room, ok := message.(outgoing.Room); ok {
				c.info.RoomID = room.ID
			}
			if err := c.conn.WriteJSON(typed); err != nil {
				conClosed()
				log.Error().Err(err).Str("clientId", c.info.ID.String()).Str("user", c.info.AuthenticatedUser).Interface("event", typed.Type).Msg("Could not write message to conn")
			}
			log.Debug().Str("clientId", c.info.ID.String()).Str("user", c.info.AuthenticatedUser).Interface("event", typed.Type).Msg("WebSocket send")
		case <-pingTicker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
			err := c.conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				conClosed()
				log.Error().Err(err).Str("clientId", c.info.ID.String()).Str("user", c.info.AuthenticatedUser).Msg("Could not write ping message")
				return
			}
		}
	}
}
