package socks

import (
	"io"
	"fmt"
	"encoding/binary"

	"../../network"
)

func canNotBeIgnoredErr(err error) bool {
	return (err != nil) && (err != io.EOF)
}

const (
	socksVersion	= byte(0x05)

	authNotRequired	= byte(0x00)
	authGssApi		= byte(0x01)
	authUserPass	= byte(0x02)
	authNoAcceptableMethod	= byte(0xFF)
)

// first, client request to build the link
type socks5AuthenticationRequest struct {
	version				byte
	nMethods			byte
	supportedMethods	[256]byte	// methods client support
}

func (r socks5AuthenticationRequest) hasSupportedMethod(method byte) bool {
	for i := 0; i < int(r.nMethods); i++ {
		if method == r.supportedMethods[i] {
			return true
		}
	}
	return false
}

func readAuthentication(reader io.Reader) (request socks5AuthenticationRequest, err error) {
	buffer := make([]byte, 256)
	nBytes, err := reader.Read(buffer)
	if canNotBeIgnoredErr(err) {
		err = fmt.Errorf("Faild to read socks auth: %v", err)
		return
	}
	if nBytes < 2 {
		err = fmt.Errorf("Expect at least 2 bytes read, but actually %d bytes read\n", nBytes)
		return
	}

	request.version = buffer[0]
	if request.version != socksVersion {
		err = fmt.Errorf("Unsupported socks version: %d", request.version)
		return
	}

	request.nMethods = buffer[1]
	if request.nMethods <= 0 {
		err = fmt.Errorf("Client has no supported methods")
		return
	}

	if nBytes - 2 != int(request.nMethods) {
		err = fmt.Errorf("Unmatching number of auth methods, expecting %d, but got %d", request.nMethods, nBytes - 2)
		return
	}

	copy(request.supportedMethods[:], buffer[2:nBytes])
	return
}

// second, server choose a auth method
type byteSlicer interface {
	byteSlice() []byte
}

func writeResponse(writer io.Writer, data byteSlicer) error {
	_, err := writer.Write(data.byteSlice())
	return err
}

type socks5AuthenticationResponse struct {
	version	byte
	method	byte
}

func (r socks5AuthenticationResponse) byteSlice() []byte {
	return []byte{r.version, r.method}
}

func newAuthenticationResponse(method byte) socks5AuthenticationResponse {
	return socks5AuthenticationResponse{
		version: socksVersion,
		method: method,
	}
}

// choose use password auth method
type socks5UserPassRequest struct {
	version 	byte
	username	string
	password	string
}

func readUserPass(reader io.Reader) (request socks5UserPassRequest, err error) {
	buffer := make([]byte, 256)
	nBytes, err := reader.Read(buffer[0:2])
	if err != nil {
		return
	}

	request.version = buffer[0]
	usernameLen := int(buffer[1])
	nBytes, err = reader.Read(buffer)
	if err != nil {
		return
	}
	if nBytes != usernameLen {
		err = fmt.Errorf("Expect %d bytes username, but got %d", usernameLen, nBytes)
		return
	}
	request.username = string(buffer[:usernameLen])

	_, err = reader.Read(buffer[:1])
	if err != nil {
		return
	}
	passwordLen := int(buffer[0])
	nBytes, err = reader.Read(buffer)
	if canNotBeIgnoredErr(err) {
		return
	}
	if nBytes != passwordLen {
		err = fmt.Errorf("Expect %d bytes password, but got %d", passwordLen, nBytes)
		return
	}
	request.password = string(buffer[:passwordLen])

	return
}

// server authenticates
type socks5UserPassResponse struct {
	version byte
	status	byte
}

func (r socks5UserPassResponse) byteSlice() []byte {
	return []byte{r.version, r.status}
}

func newUserPassResponse(status byte) socks5UserPassResponse {
	return socks5UserPassResponse{
		version: socksVersion,
		status: status,
	}
}

const (
	validUser	= byte(iota)
	invalidUser
)

// TODO
func (r socks5UserPassRequest) verifyUser() byte {
	return validUser
}

// third, client show the destination address
const (
	addrTypeIPv4   = byte(0x01)
	addrTypeIPv6   = byte(0x04)
	addrTypeDomain = byte(0x03)

	cmdConnect		= byte(0x01)
	cmdBind			= byte(0x02)
	cmdUDPAssociate	= byte(0x03)
)

type socks5ConfirmDestinationRequest struct {
	version		byte
	command		byte
	addrType	byte
	ipv4		[4]byte
	ipv6		[16]byte
	domain		string
	port		uint16
}

func readDestination(reader io.Reader) (request socks5ConfirmDestinationRequest, err error) {
	buffer := make([]byte, 4)
	nBytes, err := reader.Read(buffer)
	if err != nil {
		return
	}

	request.version  = buffer[0]
	request.command  = buffer[1]
	request.addrType = buffer[3]

	switch request.addrType {
	case addrTypeIPv4:
		nBytes, err = reader.Read(request.ipv4[:])
		if err != nil {
			return
		}
		if nBytes != len(request.ipv4) {
			err = fmt.Errorf("Failed to read complete IPv4 address")
			return
		}
	case addrTypeIPv6:
		nBytes, err = reader.Read(request.ipv6[:])
		if err != nil {
			return
		}
		if nBytes != len(request.ipv6) {
			err = fmt.Errorf("Failed to read complete IPv6 address")
			return
		}
	case addrTypeDomain:
		tmpBuffer := make([]byte, 256)
		_, err = reader.Read(tmpBuffer[:1])
		if err != nil {
			return
		}
		
		domainLen := int(tmpBuffer[0])
		nBytes, err = reader.Read(tmpBuffer[:domainLen])
		if err != nil {
			return
		}
		if nBytes != domainLen {
			err = fmt.Errorf("Expect %d bytes domain, but got %d", domainLen, nBytes)
			return
		}
		request.domain = string(tmpBuffer[:domainLen])
	default:
		err = fmt.Errorf("Unknown address type: %d", request.addrType)
		return
	}

	nBytes, err = reader.Read(buffer[:2])
	if canNotBeIgnoredErr(err) {
		return
	}
	if nBytes != 2 {
		err = fmt.Errorf("Failed to read complete destination port")
		return
	}
	request.port = binary.BigEndian.Uint16(buffer[:2])
	return
}

// fourth, server reply
const (
	statusSucceed = byte(iota)
	statusGeneralFailure
	statusConnectionNotAllowed
	statusNetworkUnreachable
	statusHostUnreachable
	statusConnectionRefused
	statusTTLExpired
	statusCommandNotSupported
	statusAddressTypeNotSupported
)

type socks5ConfirmDestinationResponse struct {
	version 	byte
	statusCode	byte
	addrType	byte
	ipv4		[4]byte
	ipv6		[16]byte
	domain		string
	port		uint16
}

func newConfirmDestinationResponse(r socks5ConfirmDestinationRequest) socks5ConfirmDestinationResponse {
	return socks5ConfirmDestinationResponse{
		version: 	r.version,
		statusCode:	statusSucceed,
		addrType:	r.addrType,
		ipv4:		r.ipv4,
		ipv6:		r.ipv6,
		domain:		r.domain,
		port:		r.port,
	}
}

func (r socks5ConfirmDestinationResponse) byteSlice() []byte {
	buffer := make ([]byte, 0, 300)
	buffer = append(buffer, r.version, r.statusCode, byte(0x00), r.addrType)

	switch r.addrType {
	case addrTypeIPv4:
		buffer = append(buffer, r.ipv4[:]...)
	case addrTypeIPv6:
		buffer = append(buffer, r.ipv6[:]...)
	case addrTypeDomain:
		buffer = append(buffer, byte(len(r.domain)))
		buffer = append(buffer, []byte(r.domain)...)
	}

	portBuffer := make([]byte, 2)
	binary.BigEndian.PutUint16(portBuffer, r.port)
	buffer = append(buffer, portBuffer...)
	return buffer
}

// ignore udp connection for the time being
func (r socks5ConfirmDestinationResponse) Destination() (dest network.Destination, err error) {
	var addr network.Address
	switch r.addrType {
	case addrTypeIPv4:
		addr, err = network.NewIPv4Address(r.ipv4[:], r.port)
		if err != nil {
			return nil, err
		}
	case addrTypeIPv6:
		addr, err = network.NewIPv6Address(r.ipv6[:], r.port)
		if err != nil {
			return nil, err
		}
	case addrTypeDomain:
		addr = network.NewDomainAddress(r.domain, r.port)
	}

	return network.NewTCPDestination(addr), nil
}