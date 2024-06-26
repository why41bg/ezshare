package ws

import (
	"fmt"
	"github.com/rs/zerolog/log"
)

func init() {
	register("join", func() Event {
		return &Join{}
	})
}

type Join struct {
	RoomID   string `json:"id"`
	UserName string `json:"username,omitempty"`
}

func (e *Join) Execute(rooms *Rooms, current ClientInfo) error {
	if current.RoomID != "" {
		return fmt.Errorf("cannot join room, you are already in one")
	}
	room, ok := rooms.Rooms[e.RoomID]
	if !ok {
		return fmt.Errorf("room with id %s does not exist", e.RoomID)
	}
	var name string
	if current.Authenticated {
		name = current.AuthenticatedUser
	} else {
		name = rooms.RandUserName()
	}

	room.Users[current.ID] = &User{
		ID:        current.ID,
		Name:      name,
		Streaming: false,
		Owner:     false,
		Addr:      current.Addr,
		Write:     current.Write,
		Close:     current.Close,
	}
	room.notifyInfoChanged()

	v4, v6, err := rooms.config.TurnIPProvider.Get()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get turn ip")
		return err
	}

	for _, user := range room.Users {
		if current.ID == user.ID || !user.Streaming {
			continue
		}
		room.newSession(user.ID, current.ID, rooms, v4, v6)
	}

	return nil
}
