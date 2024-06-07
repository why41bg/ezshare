package turn

import (
	"errors"
	"net"
	"strconv"
)

type RelayAddressGeneratorNone struct{}

func (r *RelayAddressGeneratorNone) Validate() error {
	return nil
}

func (r *RelayAddressGeneratorNone) AllocatePacketConn(network string, requestedPort int) (net.PacketConn, net.Addr, error) {
	packetConn, err := net.ListenPacket("udp", ":"+strconv.Itoa(requestedPort))
	if err != nil {
		return nil, nil, err
	}

	relayAddr := packetConn.LocalAddr().(*net.UDPAddr)
	return packetConn, relayAddr, nil
}

func (r *RelayAddressGeneratorNone) AllocateConn(network string, requestedPort int) (net.Conn, net.Addr, error) {
	return nil, nil, errors.New("tcp not supported by none relay address generator")
}
