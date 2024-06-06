package ws

import (
	"fmt"
)

func init() {
	register("share", func() Event {
		return &StartShare{}
	})
}

type StartShare struct{}

// Execute firstly checks if the user is in a room and the room is valid or not.
// Passing the checks, it sets the user's streaming status and gets the TURN server
// IP addresses. Then, it notifies all users in the room that the user is streaming.
// Lastly, it notifies the room that the user's information has changed.
func (e *StartShare) Execute(rooms *Rooms, current ClientInfo) error {
	if current.RoomID == "" {
		return fmt.Errorf("not in a room")
	}
	room, ok := rooms.Rooms[current.RoomID]
	if !ok {
		return fmt.Errorf("room with id %s does not exist", current.RoomID)
	}
	room.Users[current.ID].Streaming = true

	v4, v6, err := rooms.config.TurnIPProvider.Get()
	if err != nil {
		return err
	}

	for _, user := range room.Users {
		if current.ID == user.ID {
			continue
		}
		room.newSession(current.ID, user.ID, rooms, v4, v6)
	}

	room.notifyInfoChanged()
	return nil
}
