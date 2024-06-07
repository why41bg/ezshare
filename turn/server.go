package turn

import (
	"github.com/ezshare/server/config"
	"github.com/ezshare/server/config/ip"
	"github.com/pion/turn/v2"
	"github.com/rs/zerolog/log"
	"net"
	"sync"
)

// Server defines the interface for managing TURN servers, mainly for managing the permissions of users accessing
// the TURN server.
type Server interface {
	Credentials(id string, addr net.IP) (string, string)
	Ban(username string)
}

// InternalServer is an internal TURN server, stores user information for access control.
type InternalServer struct {
	lock   sync.RWMutex
	Lookup map[string]User
}

// User is the user information for accessing the TURN server.
type User struct {
	Addr     net.IP
	Password []byte
}

// Generator is a customer relay address generator, which can generate relay addresses based on the IP address.
type Generator struct {
	turn.RelayAddressGenerator
	IPProvider ip.Provider
}

// AllocatePacketConn allocates a PacketConn (UDP) RelayAddress.
func (r *Generator) AllocatePacketConn(network string, requestedPort int) (net.PacketConn, net.Addr, error) {
	conn, addr, err := r.RelayAddressGenerator.AllocatePacketConn(network, requestedPort)
	if err != nil {
		return conn, addr, err
	}
	relayAddr := *addr.(*net.UDPAddr)

	v4, v6, err := r.IPProvider.Get()
	if err != nil {
		return conn, addr, err
	}

	if v6 == nil || (relayAddr.IP.To4() != nil && v4 != nil) {
		relayAddr.IP = v4
	} else {
		relayAddr.IP = v6
	}

	log.Debug().
		Str("addr", addr.String()).
		Str("relayaddr", relayAddr.String()).
		Msg("TURN allocated")
	return conn, &relayAddr, err
}

// Start starts a TURN server.
func Start(config *config.Config) (Server, error) {
	udpListener, err := net.ListenPacket("udp", config.TurnAddress)
	if err != nil {
		return nil, err
	}
	tcpListener, err := net.Listen("tcp", config.TurnAddress)
	if err != nil {
		return nil, err
	}

	srv := &InternalServer{Lookup: map[string]User{}}
	gen := &Generator{
		RelayAddressGenerator: generator(*config),
		IPProvider:            config.TurnIPProvider,
	}

	_, err = turn.NewServer(turn.ServerConfig{
		Realm:       config.TurnRealm,
		AuthHandler: srv.authenticate,
		ListenerConfigs: []turn.ListenerConfig{
			{Listener: tcpListener, RelayAddressGenerator: gen},
		},
		PacketConnConfigs: []turn.PacketConnConfig{
			{PacketConn: udpListener, RelayAddressGenerator: gen},
		},
	})
	if err != nil {
		return nil, err
	}
	log.Debug().Str("address", config.TurnAddress).Msg("Started TURN server")

	return srv, nil
}

// generator returns a RelayAddressGenerator.
func generator(conf config.Config) turn.RelayAddressGenerator {
	minport, maxport, ok := conf.PortRange()
	if ok {
		log.Debug().Uint16("min", minport).Uint16("max", maxport).Msg("Using Port Range")
		return &RelayAddressGeneratorPortRange{MinPort: minport, MaxPort: maxport}
	}
	log.Debug().Msg("Using None Port Range")
	return &RelayAddressGeneratorNone{}
}

// Credentials registers a user with a random password for the given
// username and address, and returns the username and password.
func (s *InternalServer) Credentials(id string, addr net.IP) (string, string) {
	pass := "password" // TODO random password
	s.lock.Lock()
	defer s.lock.Lock()
	s.Lookup[id] = User{Addr: addr, Password: []byte(pass)}

	return id, pass
}

// Ban bans a user from using the TURN server.
func (s *InternalServer) Ban(username string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.Lookup, username)
}

// authenticate according to the given username and address to check if the user is allowed to access the TURN server.
// If the user is allowed, return the password; otherwise, return false.
func (s *InternalServer) authenticate(username, realm string, addr net.Addr) ([]byte, bool) {
	s.lock.RLock()
	defer s.lock.RLock()
	entry, ok := s.Lookup[username]
	if !ok {
		log.Debug().Str("username", username).Str("address", addr.String()).Msg("Unauthorized")
		return nil, false
	}
	return entry.Password, true
}
