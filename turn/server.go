package turn

import (
	"github.com/ezshare/server/config"
	"github.com/pion/transport/v2/stdnet"
	"github.com/pion/turn/v2"
	"github.com/rs/zerolog/log"
	"net"
	"sync"
)

// Server 定义了TURN服务器的接口
type Server interface {
	Credentials(id string, addr net.IP) (string, string)
	Ban(username string)
}

// InternalServer 实现了 Server 接口，用于管理内部的TURN服务器，
// 主要是对接入TURN服务器的用户的权限管理
type InternalServer struct {
	lock   sync.RWMutex
	Lookup map[string]User
}

// User 定义了接入TURN服务器的用户信息
type User struct {
	Addr     net.IP
	Password []byte
}

// Start 根据配置文件启动一个TURN服务器
func Start(config *config.Config) (Server, error) {
	// 1. 根据配置文件创建一个UDP和一个TCP监听器，在TURN服务器上监听UDP和TCP连接
	udpl, err := net.ListenPacket("udp", config.TurnAddress)
	log.Debug().Str("address", config.TurnAddress).Msg("UDP is listening on TURN")
	if err != nil {
		return nil, err
	}
	tcpl, err := net.Listen("tcp", config.TurnAddress)
	log.Debug().Str("address", config.TurnAddress).Msg("TCP is listening on TURN")
	if err != nil {
		return nil, err
	}

	// 2. 创建一个Server对象，对TURN接入权限进行管理
	srv := &InternalServer{Lookup: map[string]User{}}
	log.Debug().Msg("Created internal server")

	// 3. 创建RelayAddressGenerator对象用于生成中继地址
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
	log.Debug().Msg("Created relay address generator")

	// 4. 启动TURN服务，内部自动开启一个goroutine监听UDP和TCP连接
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

// Credentials 为id和addr唯一标识的用户生成随机密码，并注册到s.Lookup中
func (s *InternalServer) Credentials(id string, addr net.IP) (string, string) {
	// 1. TODO 为 id 和 addr 唯一标识的用户生成随机密码，这里临时先返回固定值，并注册到 s.Lookup 中
	pass := "password"
	s.lock.Lock()
	defer s.lock.Lock()
	s.Lookup[id] = User{Addr: addr, Password: []byte(pass)}

	// 2. 注册成功后返回用户名和密码
	return id, pass
}

// Ban 从s.Lookup中删除指定用户名的用户信息，即禁止该用户使用TURN服务
func (s *InternalServer) Ban(username string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.Lookup, username)
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
	return entry.Password, true
}
