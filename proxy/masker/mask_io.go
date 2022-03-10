package masker

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	cryptrand "crypto/rand"
	mrand "math/rand"

	"masker/account"
	"masker/cryption"
	"masker/network"
)

const (
	addrTypeIPv4   = byte(0x01)
	addrTypeIPv6   = byte(0x03)
	addrTypeDomain = byte(0x02)
)

type maskRequest struct {
	userID         *account.ID
	requestKey     [16]byte
	requestIV      [16]byte
	responseHeader [4]byte
	dest           network.Destination
}

func newMaskRequest(u account.User, dest network.Destination) *maskRequest {
	r := &maskRequest{
		userID: u.Id,
		dest:   dest,
	}
	cryptrand.Read(r.requestKey[:])
	cryptrand.Read(r.requestIV[:])
	cryptrand.Read(r.responseHeader[:])

	return r
}

func readMaskRequest(reader io.Reader, userSet account.UserSet) (request *maskRequest, err error) {
	request = new(maskRequest)
	buffer := make([]byte, 256)

	// user hash
	nBytes, err := reader.Read(buffer[:account.IDBytesLen])
	if err != nil {
		return
	}
	if nBytes != account.IDBytesLen {
		err = fmt.Errorf("Unable read complete user hash, want %d, get %d", account.IDBytesLen, nBytes)
		return
	}

	userHash := buffer[:nBytes]
	userID, timeSec, ok := userSet.GetUser(userHash)
	if ok == false {
		err = fmt.Errorf("invalid user")
		return
	}
	request.userID = userID
	decryptReader, err := cryption.NewAESDecryptReader(reader, userID.CmdKey(), cryption.Int64Hash(timeSec))
	if err != nil {
		return
	}

	// skip random padding
	skipRandomPadding := func() error {
		_, err := decryptReader.Read(buffer[:1])
		if err != nil {
			return err
		}

		randomPaddingLen := int(buffer[0])
		if randomPaddingLen <= 0 || randomPaddingLen > 32 {
			return fmt.Errorf("Unexpected random padding length %d", randomPaddingLen)
		}
		_, err = decryptReader.Read(buffer[:randomPaddingLen])
		return err
	}

	if err = skipRandomPadding(); err != nil {
		return
	}

	// key
	nBytes, err = decryptReader.Read(request.requestKey[:])
	if err != nil {
		return
	}
	if nBytes != len(request.requestKey) {
		err = fmt.Errorf("Unable read complete request key, want %d, get %d", len(request.requestKey), nBytes)
		return
	}

	// IV
	nBytes, err = decryptReader.Read(request.requestIV[:])
	if err != nil {
		return
	}
	if nBytes != len(request.requestIV) {
		err = fmt.Errorf("Unable read complete request IV, want %d, get %d", len(request.requestIV), nBytes)
		return
	}

	// response header
	nBytes, err = decryptReader.Read(request.responseHeader[:])
	if err != nil {
		return
	}
	if nBytes != len(request.responseHeader) {
		err = fmt.Errorf("Unable read complete response header, want %d, get %d", len(request.responseHeader), nBytes)
		return
	}

	// port
	nBytes, err = decryptReader.Read(buffer[:2])
	if err != nil {
		return
	}
	if nBytes != 2 {
		err = fmt.Errorf("Unable read complete port, want 2, get %d", nBytes)
	}
	port := binary.BigEndian.Uint16(buffer[0:2])

	// address
	_, err = decryptReader.Read(buffer[:1])
	if err != nil {
		return
	}
	var addr network.Address
	switch buffer[0] {
	case addrTypeIPv4:
		nBytes, err = decryptReader.Read(buffer[1:5])
		if err != nil {
			return
		}
		if nBytes != 4 {
			err = fmt.Errorf("Unable read complete ip, want 4, get %d", nBytes)
			return
		}

		addr, err = network.NewIPv4Address(buffer[1:5], port)
		if err != nil {
			return
		}
	case addrTypeIPv6:
		nBytes, err = decryptReader.Read(buffer[1:17])
		if err != nil {
			return
		}
		if nBytes != 16 {
			err = fmt.Errorf("Unable read complete ip, want 16, get %d", nBytes)
			return
		}

		addr, err = network.NewIPv6Address(buffer[1:17], port)
		if err != nil {
			return
		}
	case addrTypeDomain:
		_, err = decryptReader.Read(buffer[1:2])
		if err != nil {
			return
		}

		domainLen := int(buffer[1])
		nBytes, err = decryptReader.Read(buffer[2 : 2+domainLen])
		if err != nil {
			return
		}
		if nBytes != domainLen {
			err = fmt.Errorf("Unable read complete domain, want %d, get %d", domainLen, nBytes)
			return
		}
		addr = network.NewDomainAddress(string(buffer[2:2+domainLen]), port)
	default:
		err = fmt.Errorf("Unsupported address type: %v", buffer[0])
		return
	}
	request.dest = network.NewTCPDestination(addr)

	// skip random padding
	if err = skipRandomPadding(); err != nil {
		return
	}
	return
}

func (r *maskRequest) encryptedByteSlice() ([]byte, error) {
	buffer := make([]byte, 0, 300)

	// add random padding
	randomPadding := func() error {
		randomPaddingLen := mrand.Intn(32) + 1
		randomPaddingContent := make([]byte, randomPaddingLen)
		_, err := mrand.Read(randomPaddingContent)
		if err != nil {
			return err
		}

		buffer = append(buffer, byte(randomPaddingLen))
		buffer = append(buffer, randomPaddingContent...)
		return nil
	}

	if err := randomPadding(); err != nil {
		return nil, err
	}

	// add key, IV
	// TODO: add user hash
	buffer = append(buffer, r.requestKey[:]...)
	buffer = append(buffer, r.requestIV[:]...)
	buffer = append(buffer, r.responseHeader[:]...)

	// add dest address
	buffer = append(buffer, r.dest.PortByteSlice()...)

	switch {
	case r.dest.IsIPv4():
		buffer = append(buffer, addrTypeIPv4)
		buffer = append(buffer, r.dest.IP()...)
	case r.dest.IsIPv6():
		buffer = append(buffer, addrTypeIPv6)
		buffer = append(buffer, r.dest.IP()...)
	case r.dest.IsDomain():
		buffer = append(buffer, addrTypeDomain)
		domain := []byte(r.dest.Domain())
		buffer = append(buffer, byte(len(domain)))
		buffer = append(buffer, domain...)
	}

	// add random padding
	if err := randomPadding(); err != nil {
		return nil, err
	}

	// add user hash and encrypt request header
	// random time: +-30s from current time
	randomTimeSec := time.Now().Unix() - 30 + mrand.Int63n(61)
	userHash := cryption.TimeHMACHash(r.userID.Bytes, randomTimeSec)

	aesEncryptStream, err := cryption.NewAESEncryptStream(r.userID.CmdKey(), cryption.Int64Hash(randomTimeSec))
	if err != nil {
		return nil, err
	}
	aesEncryptStream.XORKeyStream(buffer, buffer)

	buffer = append(userHash, buffer...)
	return buffer, nil
}

type maskResponse [4]byte

func newMaskResponse(request *maskRequest) *maskResponse {
	response := new(maskResponse)
	copy(response[:], request.responseHeader[:])
	return response
}
