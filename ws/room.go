package ws

import (
	"fmt"
	"github.com/ezshare/server/ws/outgoing"
	"github.com/rs/xid"
	"net"
	"sort"
)

type ConnectionMode string

const (
	ConnectionLocal ConnectionMode = "local"
	ConnectionSTUN  ConnectionMode = "stun"
	ConnectionTURN  ConnectionMode = "turn"
)

type Room struct {
	ID                string
	CloseOnOwnerLeave bool
	ConnectionMode    ConnectionMode
	Users             map[xid.ID]*User
	Sessions          map[xid.ID]*RoomSession
}

type User struct {
	ID        xid.ID // Client ID
	Addr      net.IP // Client IP address
	Name      string // If the client is authenticated, it is the authenticated username, otherwise it is a random username
	Streaming bool
	Owner     bool
	Write     chan<- outgoing.Message // Client write channel which to send messages to the client
	Close     chan<- string           // Client close channel which to send a close signal to the client
}

// RoomSession here has a stream channel from the Host to the Client.
type RoomSession struct {
	Host   xid.ID
	Client xid.ID
}

const (
	CloseOwnerLeft = "Owner Left"
	CloseDone      = "Read End"
)

// newSession creates a new session between the host and the client. The host and client are the
// ClientInfo.ID. The v4 and v6 are the IP addresses of the TURN server.
func (r *Room) newSession(host, client xid.ID, rooms *Rooms, v4, v6 net.IP) {
	id := xid.New()
	r.Sessions[id] = &RoomSession{
		Host:   host,
		Client: client,
	}

	var iceHost []outgoing.ICEServer
	var iceClient []outgoing.ICEServer
	switch r.ConnectionMode {
	case ConnectionLocal:
	case ConnectionSTUN:
		iceHost = []outgoing.ICEServer{{URLs: rooms.addresses("stun", v4, v6, false)}}
		iceClient = []outgoing.ICEServer{{URLs: rooms.addresses("stun", v4, v6, false)}}
	case ConnectionTURN:
		hostName, hostPW := rooms.turnServer.Credentials(id.String()+"host", r.Users[host].Addr)
		clientName, clientPW := rooms.turnServer.Credentials(id.String()+"client", r.Users[client].Addr)
		iceHost = []outgoing.ICEServer{{
			URLs:       rooms.addresses("turn", v4, v6, true),
			Credential: hostPW,
			Username:   hostName,
		}}
		iceClient = []outgoing.ICEServer{{
			URLs:       rooms.addresses("turn", v4, v6, true),
			Credential: clientPW,
			Username:   clientName,
		}}
	}
	r.Users[host].Write <- outgoing.HostSession{Peer: client, ID: id, ICEServers: iceHost}
	r.Users[client].Write <- outgoing.ClientSession{Peer: host, ID: id, ICEServers: iceClient}
}

// addresses generates the STUN or TURN server address for the given IP.
func (r *Rooms) addresses(prefix string, v4, v6 net.IP, tcp bool) (result []string) {
	if v4 != nil {
		result = append(result, fmt.Sprintf("%s:%s:%s", prefix, v4.String(), r.config.TurnPort))
		if tcp {
			result = append(result, fmt.Sprintf("%s:%s:%s?transport=tcp", prefix, v4.String(), r.config.TurnPort))
		}
	}
	if v6 != nil {
		result = append(result, fmt.Sprintf("%s:[%s]:%s", prefix, v6.String(), r.config.TurnPort))
		if tcp {
			result = append(result, fmt.Sprintf("%s:[%s]:%s?transport=tcp", prefix, v6.String(), r.config.TurnPort))
		}
	}
	return
}

// closeSession closes the session between the host and the client. If the connection mode is TURN,
// the TURN server is informed to ban the host and the client from the TURN server.
func (r *Room) closeSession(rooms *Rooms, id xid.ID) {
	if r.ConnectionMode == ConnectionTURN {
		rooms.turnServer.Ban(id.String() + "host")
		rooms.turnServer.Ban(id.String() + "client")
	}
	delete(r.Sessions, id)
}

// notifyInfoChanged loops over all users in the room and sends them the updated room information.
func (r *Room) notifyInfoChanged() {
	for _, current := range r.Users {
		var users []outgoing.User
		for _, user := range r.Users {
			users = append(users, outgoing.User{
				ID:        user.ID,
				Name:      user.Name,
				Streaming: user.Streaming,
				You:       current == user,
				Owner:     user.Owner,
			})
		}

		sort.Slice(users, func(i, j int) bool {
			left := users[i]
			right := users[j]
			if left.Owner != right.Owner {
				return left.Owner
			}
			if left.Streaming != right.Streaming {
				return left.Streaming
			}
			return left.Name < right.Name
		})

		current.Write <- outgoing.Room{
			ID:    r.ID,
			Users: users,
		}
	}
}
