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
	conn    *websocket.Conn
	info    ClientInfo
	once    once
	toRooms chan<- ClientMessage
}

// ClientInfo contains the information of a client.
type ClientInfo struct {
	ID                xid.ID // A unique ID for the client
	RoomID            string // The room which the client is in
	Authenticated     bool   // The creator of the client is authenticated or not
	AuthenticatedUser string // If not authenticated, "guest"
	Write             chan outgoing.Message
	Close             chan string
	Addr              net.IP
}

// ClientMessage describes an event received from a client and the client's information.
type ClientMessage struct {
	Info     ClientInfo
	Incoming Event
}

// newClient creates a new Client to wrap a websocket connection. And set the close handler for the connection.
// It returns the reference of the created Client object.
func newClient(conn *websocket.Conn, read chan ClientMessage, authenticatedUser string, authenticated bool) *Client {
	// 创建一个新的Client对象包装WebSocket连接，并设置其关闭时的回调函数
	c := &Client{
		conn: conn, // Websocket connection
		info: ClientInfo{
			ID:                xid.New(),                           // A unique ID for the client
			RoomID:            "",                                  // The room which the client is in
			Authenticated:     authenticated,                       // The creator of the client is authenticated or not
			AuthenticatedUser: authenticatedUser,                   // If authenticated is false, it is "guest"
			Write:             make(chan outgoing.Message, 1),      // The channel to send messages to the websocket
			Close:             make(chan string, 1),                // The channel to send a clos signal to the websocket
			Addr:              conn.RemoteAddr().(*net.TCPAddr).IP, // The IP address of the client
		},
		toRooms: read, // The channel to send messages which received from the websocket to the Rooms
	}
	log.Debug().Str("clientId", c.info.ID.String()).Str("user", c.info.AuthenticatedUser).Msg("New client created")
	conn.SetCloseHandler(func(code int, text string) error {
		message := websocket.FormatCloseMessage(code, text)
		log.Debug().Str("clientId", c.info.ID.String()).Str("reason", text).Int("code", code).Msg("WebSocket Close")
		return conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(WriteTimeout))
	})
	return c
}

// Close closes the websocket connection and sends a message to Rooms.
func (c *Client) Close() {
	c.once.Do(func() {
		_ = c.conn.Close()
		log.Debug().
			Str("clientId", c.info.ID.String()).
			Str("user", c.info.AuthenticatedUser).
			Msg("WebSocket Close")
		go func() {
			c.toRooms <- ClientMessage{
				Info:     c.info,
				Incoming: &Disconnected{},
			}
		}()
	})
}

// startReading try to get the next reader from the websocket connection. If the message type
// is websocket.BinaryMessage, close the connection. If the message is websocket.TextMessage,
// then parse it and send it to the Rooms.
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
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Str("clientId", c.info.ID.String()).Str("user", c.info.AuthenticatedUser).Msg("Unexpected close error")
			} else {
				log.Debug().Str("clientId", c.info.ID.String()).Str("user", c.info.AuthenticatedUser).Msg("Close reader")
			}
			return
		}

		if t == websocket.BinaryMessage {
			_ = c.conn.CloseHandler()(websocket.CloseUnsupportedData, fmt.Sprintf("Unsupported binary message type"))
			return
		}
		event, err := ParseTypedIncoming(m)
		if err != nil {
			_ = c.conn.CloseHandler()(websocket.CloseNormalClosure, fmt.Sprintf("Failed to parse message: %s", err))
			return
		}
		c.toRooms <- ClientMessage{Info: c.info, Incoming: event}
	}
}

// startWriteHandler reads messages from the write channel and sends them to the
// websocket connection. It also sends ping to the connection at regular intervals.
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
			log.Debug().Str("clientId", c.info.ID.String()).Str("user", c.info.AuthenticatedUser).Interface("event", typed.Type).Msg("Send a message to client successfully")
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
