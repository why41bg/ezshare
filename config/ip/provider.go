package ip

import "net"

type Provider interface {
	Get() (v4, v6 net.IP, err error)
}
