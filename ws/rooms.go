package ws

import (
	"fmt"
	"github.com/ezshare/server/auth"
	"github.com/ezshare/server/config"
	"github.com/ezshare/server/turn"
	"github.com/ezshare/server/util"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"math/rand"
	"net/http"
	"net/url"
	"reflect"
	"time"
)

type Rooms struct {
	turnServer turn.Server
	Rooms      map[string]*Room   // RoomID -> Room
	Incoming   chan ClientMessage // Receive messages from clients. All clients send messages to this channel.
	upgrader   websocket.Upgrader // The function to upgrade an HTTP request to a WebSocket connection.
	users      *auth.Users        // Loaded user information from the user file in local.
	config     config.Config
	r          *rand.Rand
}

// NewRooms creates a new Rooms object and define the function to upgrade an HTTP request to a WebSocket
// connection. Return the reference of the created Rooms object.
//
// This function only runs once when the server starts.
func NewRooms(turnServer turn.Server, users *auth.Users, conf config.Config) *Rooms {
	log.Debug().Msg("Creating rooms")
	return &Rooms{
		Rooms:      map[string]*Room{},
		Incoming:   make(chan ClientMessage),
		turnServer: turnServer,
		users:      users,
		config:     conf,
		r:          rand.New(rand.NewSource(time.Now().Unix())),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("origin")
				u, err := url.Parse(origin)
				if err != nil {
					return false
				}
				if u.Host == r.Host {
					return true
				}
				return conf.CheckOrigin(origin)
			},
		},
	}
}

// Upgrade upgrades an HTTP request to a websocket connection. And wrap the websocket connection
// with a Client object.
//
// Lastly, start two goroutines, one to read messages from websocket and them to Rooms, the other
// write messages which received from Rooms to websocket.
func (r *Rooms) Upgrade(w http.ResponseWriter, req *http.Request) {
	ws, err := r.upgrader.Upgrade(w, req, nil)
	log.Debug().Str("remoteAddr", req.RemoteAddr).Msg("Upgrade to websocket")
	if err != nil {
		log.Error().Err(err).Msg("Upgrade failed")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(fmt.Sprintf("Upgrade failed %s", err)))
		return
	}

	user, loggedIn := r.users.CurrentUser(req)
	c := newClient(ws, r.Incoming, user, loggedIn)
	go c.startReading(time.Second * 20)
	log.Debug().Str("clientId", c.info.ID.String()).Str("user", c.info.AuthenticatedUser).Msg("Start reading from websocket")
	go c.startWriteHandler(time.Second * 5)
	log.Debug().Str("clientId", c.info.ID.String()).Str("user", c.info.AuthenticatedUser).Msg("Start writing to websocket")
}

// Start listens on the Incoming channel and executes the Incoming.
func (r *Rooms) Start() {
	for {
		msg := <-r.Incoming
		log.Debug().
			Str("clientId", msg.Info.ID.String()).
			Str("user", msg.Info.AuthenticatedUser).
			Str("event", reflect.TypeOf(msg.Incoming).Elem().Name()).
			Interface("eventInfo", msg.Incoming).
			Msg("Server received a message from client")

		if err := msg.Incoming.Execute(r, msg.Info); err != nil {
			log.Error().Err(err).Msg("Failed to execute Incoming message")
			msg.Info.Close <- err.Error()
		}
	}
}

// closeRoom closes a room. First it closes all sessions in the room, then it
// deletes the room.
func (r *Rooms) closeRoom(roomID string) {
	room, ok := r.Rooms[roomID]
	if !ok {
		log.Error().Str("id", roomID).Msg("Not found room to close")
		return
	}
	for id := range room.Sessions {
		room.closeSession(r, id)
		log.Debug().Str("roomId", roomID).Str("sessionId", id.String()).Msg("Close session")
	}
	delete(r.Rooms, roomID)
	log.Debug().Str("roomId", roomID).Msg("Room closed")
}

// RandUserName generates a random username.
func (r *Rooms) RandUserName() string {
	return util.NewUserName(r.r)
}

// RandRoomName generates a random room name.
func (r *Rooms) RandRoomName() string {
	return util.NewRoomName(r.r)
}
