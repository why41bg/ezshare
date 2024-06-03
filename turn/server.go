package turn

import (
	"github.com/ezshare/server/config"
	"github.com/pion/transport/v2/stdnet"
	"github.com/pion/turn/v2"
	"github.com/rs/zerolog/log"
	"net"
	"sync"
)

type Server interface {
	Credentials(id string, addr net.IP) (string, string)
	Ban(username string)
}

// InternalServer 用于管理内部的TURN服务器
type InternalServer struct {
	lock   sync.RWMutex
	Lookup map[string]User
}

type User struct {
	Addr     net.IP
	Password []byte
}

// Start 根据配置文件启动一个TURN服务器
func Start(config *config.Config) (Server, error) {
	// 根据配置文件生成一个TURN服务器实例
	udpl, err := net.ListenPacket("udp", config.TurnAddress)
	if err != nil {
		return nil, err
	}
	tcpl, err := net.Listen("tcp", config.TurnAddress)
	if err != nil {
		return nil, err
	}

	srv := &InternalServer{Lookup: map[string]User{}}
	nt, err := stdnet.NewNet()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create network")
		return nil, err
	}
	gen := &turn.RelayAddressGeneratorStatic{
		RelayAddress: net.IP(config.TurnAddress),
		Address:      config.TurnAddress,
		Net:          nt,
	}
	// NewServer在内部运行一个goroutine来启动TURN服务，无需返回
	_, err = turn.NewServer(turn.ServerConfig{
		Realm:       config.TurnRealm,
		AuthHandler: srv.authenticate,
		ListenerConfigs: []turn.ListenerConfig{
			{Listener: tcpl, RelayAddressGenerator: gen},
		},
		PacketConnConfigs: []turn.PacketConnConfig{
			{PacketConn: udpl, RelayAddressGenerator: gen},
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to start TURN server")
		return nil, err
	}

	log.Info().Str("address", config.TurnAddress).Msg("Started TURN server")
	return srv, nil
}

// Credentials 为 id 和 addr 唯一标识的用户生成随机密码，并注册到 s.Lookup 中
func (s *InternalServer) Credentials(id string, addr net.IP) (string, string) {
	// TODO 为 id 和 addr 唯一标识的用户生成随机密码，这里临时先返回固定值
	pass := "password"

	// 将用户信息注册到 s.Lookup 中
	s.lock.Lock()
	defer s.lock.Lock()
	s.Lookup[id] = User{Addr: addr, Password: []byte(pass)}
	log.Info().Str("id", id).IPAddr("addr", addr).Msg("Registered")

	// 注册成功后返回用户名和密码
	return id, pass
}

// Ban 从 s.Lookup 中删除指定用户名的用户信息，即禁止该用户使用TURN服务
func (s *InternalServer) Ban(username string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.Lookup, username)
	log.Info().Str("username", username).Msg("Banned")
}

// authenticate 查询 s.Lookup 中是否存在指定用户名的用户信息，如果存在则返回密码，否则返回 false
func (s *InternalServer) authenticate(username, realm string, addr net.Addr) ([]byte, bool) {
	// 由于 realm 固定，这里不做校验，直接查询用户信息返回即可
	s.lock.RLock()
	defer s.lock.RLock()
	entry, ok := s.Lookup[username]
	if !ok {
		log.Info().Str("username", username).Str("realm", realm).Str("address", addr.String()).Msg("Unauthorized")
		return nil, false
	}

	log.Info().Str("username", username).Str("realm", realm).Str("address", addr.String()).Msg("Authorized")
	return entry.Password, true
}
