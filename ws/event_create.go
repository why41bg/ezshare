package ws

import (
	"errors"
	"fmt"
	"github.com/ezshare/server/config"

	"github.com/rs/xid"
)

func init() {
	register("create", func() Event {
		return &Create{}
	})
}

type Create struct {
	RoomId            string `json:"id"`
	CloseOnOwnerLeave bool   `json:"closeOnOwnerLeave"`
	ConnectionMode    ConnectionMode
	UserName          string `json:"username"`
	JoinIfExist       bool   `json:"joinIfExist,omitempty"`
}

func (e *Create) Execute(rooms *Rooms, current ClientInfo) error {
	// When receiving a creation event, first check if the client is already in a room.
	if current.RoomID != "" {
		return fmt.Errorf("cannot join room, you are already in one")
	}

	// Check if the room already exists. If it does, join the existing room if the client wants to.
	if _, ok := rooms.Rooms[e.RoomId]; ok {
		if e.JoinIfExist {
			join := &Join{UserName: e.UserName, RoomID: e.RoomId}
			return join.Execute(rooms, current)
		}
		return fmt.Errorf("room with id %s does already exist", e.RoomId)
	}

	// If the room does not exist, create a new room and set the request client
	// as the owner of the room. If the client is authenticated, use its username
	// as the room owner's username, otherwise generate a random username for the guest.
	var username string
	if current.Authenticated {
		username = current.AuthenticatedUser
	} else {
		username = rooms.RandUserName()
	}

	switch rooms.config.AuthMode {
	case config.AuthModeNone:
		// Do nothing
	case config.AuthModeAll:
		// Always require authentication
		if !current.Authenticated {
			return errors.New("you need to login")
		}
	case config.AuthModeTurn:
		// Only require authentication for TURN connections
		if e.ConnectionMode == ConnectionTURN && !current.Authenticated {
			return errors.New("you need to login")
		}
	}

	room := &Room{
		ID:                e.RoomId,
		CloseOnOwnerLeave: e.CloseOnOwnerLeave,
		ConnectionMode:    e.ConnectionMode,
		Sessions:          map[xid.ID]*RoomSession{},
		Users: map[xid.ID]*User{
			current.ID: {
				ID:        current.ID,
				Name:      username,
				Streaming: false,
				Owner:     true,
				Addr:      current.Addr,
				Write:     current.Write,
				Close:     current.Close,
			},
		},
	}
	rooms.Rooms[e.RoomId] = room
	room.notifyInfoChanged()
	return nil
}
