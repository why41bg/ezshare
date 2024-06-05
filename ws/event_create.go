package ws

import (
	"fmt"

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
			join := &Join{UserName: e.UserName, ID: e.RoomId}
			return join.Execute(rooms, current)
		}
		return fmt.Errorf("room with id %s does already exist", e.RoomId)
	}

	// If the room does not exist, create a new room.
	name := rooms.RandUserName()
	if current.Authenticated {
		name = current.AuthenticatedUser
	}
	room := &Room{
		ID:                e.RoomId,
		CloseOnOwnerLeave: e.CloseOnOwnerLeave,
		Sessions:          map[xid.ID]*RoomSession{},
		Users: map[xid.ID]*User{
			current.ID: {
				ID:        current.ID,
				Name:      name,
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
