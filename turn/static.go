package turn

import (
	"github.com/pion/transport/v2"
	"net"
)

type DelayAddressGeneratorStatic struct {
	RelayAddress net.IP

	Address string

	Net transport.Net
}

func (g *DelayAddressGeneratorStatic) Validate() error {
	return nil
}

func (g *DelayAddressGeneratorStatic) AllocatePacketConn(network string, requestedPort int) (net.PacketConn, net.Addr, error) {
	return nil, nil, nil
}

func (g *DelayAddressGeneratorStatic) AllocateConn(network string, requestedPort int) (net.Conn, net.Addr, error) {
	return nil, nil, nil
}
