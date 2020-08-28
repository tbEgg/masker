package network

import (
	"net"
	"strconv"
)

const (
	_ = byte(iota)
	AddrTypeIPv4
	AddrTypeErr
	AddrTypeDomain
	AddrTypeIPv6
)

type Address struct {
	Type	byte
	IP		net.IP
	Domain	string
	Port	uint16
}

func NewIPAddress(ip []byte, port uint16) Address {
	tmpIP := make(net.IP, len(ip))
	copy(tmpIP, ip)

	if tmpIP.To4() != nil {
		return Address{
			Type:	AddrTypeIPv4,
			IP: 	tmpIP,
			Port: 	port,
		}
	} else if tmpIP.To16() != nil {
		return Address{
			Type:	AddrTypeIPv6,
			IP: 	tmpIP,
			Port: 	port,
		}
	} else {
		return Address{
			Type:	AddrTypeErr,
		}
	}
}

func NewDomainAddress(domain string, port uint16) Address {
	return Address{
		Type:	AddrTypeDomain,
		Domain:	domain,
		Port:	port,
	}
}

// ignore err type address
func (addr Address) String() string {
	host := addr.Domain
	switch addr.Type {
	case AddrTypeIPv4:
		host = addr.IP.String()
	case AddrTypeIPv6:
		host = addr.IP.String()
		host = "[" + host + "]"
	}

	return host + ":" + strconv.Itoa(int(addr.Port))
}