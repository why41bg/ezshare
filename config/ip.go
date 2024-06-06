package config

import (
	"errors"
	"github.com/ezshare/server/config/ip"
	"net"
)

func parseIPProvider(ips []string) (ip.Provider, error) {
	if len(ips) == 0 {
		return nil, errors.New("must have at least one ip")
	} else if len(ips) > 2 {
		return nil, errors.New("too many ips supplied")
	}

	static, err := parseIPStatic(ips)
	if err != nil {
		return nil, err
	}
	return static, nil
}

func parseIPStatic(ips []string) (*ip.Static, error) {
	static := &ip.Static{}
	firstIP := net.ParseIP(ips[0])
	isV4 := firstIP.To4() != nil
	if isV4 {
		static.V4 = firstIP
	} else {
		static.V6 = firstIP
	}

	if len(ips) == 1 {
		return static, nil
	}

	secondIP := net.ParseIP(ips[1])
	isV6 := secondIP.To4() == nil
	if isV4 != isV6 {
		return nil, errors.New("invalid ips: the ips must be of different type ipv4/ipv6")
	}

	if isV6 {
		static.V6 = secondIP
	} else {
		static.V4 = secondIP
	}

	return static, nil
}
