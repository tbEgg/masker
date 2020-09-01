package network

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var illegalIPv4List [][]byte = [][]byte{
	nil,
	[]byte{byte(1), byte(2), byte(3)},
	[]byte{byte(1), byte(2), byte(3), byte(4), byte(5)},
	net.ParseIP("2001:db8::68"), // ipv6
}

var port uint16 = 3333

func TestNewIPv4Address(t *testing.T) {
	for _, illegalIP := range illegalIPv4List {
		_, err := NewIPv4Address(illegalIP, port)
		if cmp.Equal(err, ErrIllegalIPv4Address, cmpopts.EquateErrors()) == false {
			t.Errorf("illegal ipv4: %s, but is accepted by mistake", string(illegalIP))
		}
	}

	ip := "1.1.1.1"
	if addr, err := NewIPv4Address(net.ParseIP(ip), port); err != nil {
		t.Errorf("legal ip: %s, but is denied by mistake", ip)
	} else {
		expectation := &AddressExpectation{
			port:         port,
			isIPv4:       true,
			expectString: "1.1.1.1:3333",
		}
		testAddress(t, addr, expectation)
	}
}

func TestNewIPv6Address(t *testing.T) {
	for _, illegalIP := range illegalIPv4List[:3] {
		_, err := NewIPv6Address(illegalIP, port)
		if cmp.Equal(err, ErrIllegalIPv6Address, cmpopts.EquateErrors()) == false {
			t.Errorf("illegal ip: %s, but is accepted by mistake", string(illegalIP))
		}
	}

	ip := "2001:db8::68"
	if addr, err := NewIPv6Address(net.ParseIP(ip), port); err != nil {
		t.Errorf("legal ip: %s, but is denied by mistake", ip)
	} else {
		expectation := &AddressExpectation{
			port:         port,
			isIPv6:       true,
			expectString: "[2001:db8::68]:3333",
		}
		testAddress(t, addr, expectation)
	}
}

func TestNewDomainAddress(t *testing.T) {
	domain := "www.baidu.com"
	addr := NewDomainAddress(domain, port)
	expectation := &AddressExpectation{
		port:         port,
		isDomain:     true,
		expectString: "www.baidu.com:3333",
	}
	testAddress(t, addr, expectation)
}

type AddressExpectation struct {
	port         uint16
	isIPv4       bool
	isIPv6       bool
	isDomain     bool
	expectString string
}

func testAddress(t *testing.T, addr Address, expectation *AddressExpectation) {
	// port
	if expectation.port != binary.BigEndian.Uint16(addr.PortByteSlice()) {
		t.Errorf("Err in Func PortByteSlice, want %d but result port is %d", expectation.port, binary.BigEndian.Uint16(addr.PortByteSlice()))
	}

	// judge address type
	actualType := []bool{addr.IsIPv4(), addr.IsIPv6(), addr.IsDomain()}
	expectType := []bool{expectation.isIPv4, expectation.isIPv6, expectation.isDomain}
	for i, actual := range actualType {
		if actual != expectType[i] {
			t.Errorf("Address type error, actutal type is %v but expect type is %v.", actualType, expectType)
			break
		}
	}

	// final result address string
	if addr.String() != expectation.expectString {
		t.Errorf("Err in get address, want %s but get %s", expectation.expectString, addr.String())
	}
}
