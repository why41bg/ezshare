package turn

import (
	"errors"
	"fmt"
	"net"

	"github.com/pion/randutil"
)

type RelayAddressGeneratorPortRange struct {
	MinPort uint16
	MaxPort uint16
	Rand    randutil.MathRandomGenerator
}

// Validate confirms that the RelayAddressGenerator is properly initialized
func (r *RelayAddressGeneratorPortRange) Validate() error {
	if r.Rand == nil {
		r.Rand = randutil.NewMathRandomGenerator()
	}
	return nil
}

// AllocatePacketConn allocates a PacketConn (UDP) RelayAddress.
func (r *RelayAddressGeneratorPortRange) AllocatePacketConn(network string, requestedPort int) (net.PacketConn, net.Addr, error) {
	if requestedPort != 0 {
		// No IP address specified, listening on requestedPort port on all network interfaces
		packetConn, err := net.ListenPacket("udp", fmt.Sprintf(":%d", requestedPort))
		if err != nil {
			return nil, nil, err
		}
		relayAddr := packetConn.LocalAddr().(*net.UDPAddr)
		return packetConn, relayAddr, nil
	}

	for try := 0; try < 10; try++ {
		port := r.MinPort + uint16(r.Rand.Intn(int((r.MaxPort+1)-r.MinPort)))
		packetConn, err := net.ListenPacket("udp", fmt.Sprintf(":%d", port))
		if err != nil {
			continue
		}
		relayAddr := packetConn.LocalAddr().(*net.UDPAddr)
		return packetConn, relayAddr, nil
	}

	return nil, nil, errors.New("could not find free port: max retries exceeded")
}

// AllocateConn allocates a Conn (TCP) RelayAddress
func (r *RelayAddressGeneratorPortRange) AllocateConn(network string, requestedPort int) (net.Conn, net.Addr, error) {
	return nil, nil, errors.New("tcp not supported by port range relay address generator")
}
