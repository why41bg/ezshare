package ws

import (
	"fmt"

	"github.com/ezshare/server/ws/outgoing"
	"github.com/rs/zerolog/log"
)

func init() {
	register("hostoffer", func() Event {
		return &HostOffer{}
	})
}

type HostOffer outgoing.P2PMessage

func (e *HostOffer) Execute(rooms *Rooms, current ClientInfo) error {
	if current.RoomID == "" {
		return fmt.Errorf("not in a room")
	}

	room, ok := rooms.Rooms[current.RoomID]
	if !ok {
		return fmt.Errorf("room with id %s does not exist", current.RoomID)
	}

	session, ok := room.Sessions[e.SID]
	if !ok {
		log.Debug().Str("sessionId", e.SID.String()).Msg("Unknown session")
		return nil
	}

	if session.Host != current.ID {
		return fmt.Errorf("permission denied for session %s", e.SID)
	}

	room.Users[session.Client].Write <- outgoing.HostOffer(*e)

	return nil
}
