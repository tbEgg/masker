package network

import (
	"errors"
	"net"
	"strconv"
)

var (
	ErrIllegalIPv4Address = errors.New("illegal ipv4 address")
	ErrIllegalIPv6Address = errors.New("illegal ipv6 address")
)

type Address interface {
	IP() net.IP
	Domain() string
	Port() uint16
	PortByteSlice() []byte

	IsIPv4() bool
	IsIPv6() bool
	IsDomain() bool

	String() string
}

func NewIPv4Address(ip []byte, port uint16) (Address, error) {
	actualIP := net.IP(ip).To4()
	if actualIP == nil {
		return nil, ErrIllegalIPv4Address
	}

	return ipv4Address{
		portAddress: portAddress(port),
		ip:          actualIP,
	}, nil
}

func NewIPv6Address(ip []byte, port uint16) (Address, error) {
	actualIP := net.IP(ip).To16()
	if actualIP == nil {
		return nil, ErrIllegalIPv6Address
	}

	return ipv6Address{
		portAddress: portAddress(port),
		ip:          actualIP,
	}, nil
}

func NewDomainAddress(domain string, port uint16) Address {
	return domainAddress{
		portAddress: portAddress(port),
		domain:      domain,
	}
}

type portAddress uint16

func (port portAddress) Port() uint16 {
	return uint16(port)
}

func (port portAddress) PortByteSlice() []byte {
	return []byte{byte(port >> 8), byte(port)}
}

func (port portAddress) IsIPv4() bool {
	return false
}

func (port portAddress) IsIPv6() bool {
	return false
}

func (port portAddress) IsDomain() bool {
	return false
}

func (port portAddress) String() string {
	return strconv.Itoa(int(port))
}

type ipv4Address struct {
	portAddress
	ip net.IP
}

func (addr ipv4Address) IP() net.IP {
	return addr.ip
}

func (addr ipv4Address) Domain() string {
	panic("calling Domain() on an ip address")
}

func (addr ipv4Address) IsIPv4() bool {
	return true
}

func (addr ipv4Address) String() string {
	return addr.ip.String() + ":" + addr.portAddress.String()
}

type ipv6Address struct {
	portAddress
	ip net.IP
}

func (addr ipv6Address) IP() net.IP {
	return addr.ip
}

func (addr ipv6Address) Domain() string {
	panic("calling Domain() on an ip address")
}

func (addr ipv6Address) IsIPv6() bool {
	return true
}

func (addr ipv6Address) String() string {
	return "[" + addr.ip.String() + "]:" + addr.portAddress.String()
}

type domainAddress struct {
	portAddress
	domain string
}

func (addr domainAddress) IP() net.IP {
	panic("calling IP() on a domain address")
}

func (addr domainAddress) Domain() string {
	return addr.domain
}

func (addr domainAddress) IsDomain() bool {
	return true
}

func (addr domainAddress) String() string {
	return addr.domain + ":" + addr.portAddress.String()
}
