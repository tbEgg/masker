package masker

import (
	"io"
	"fmt"
	"time"
	"encoding/binary"

	mrand "math/rand"
	cryptrand "crypto/rand"

	"../../network"
	"../../account"
	"../../cryption"
)

type maskRequest struct{
	userID			*account.ID
	requestKey		[16]byte
	requestIV		[16]byte
	responseHeader	[4]byte
	dest			network.Address
}

func newMaskRequest(u account.User, dest network.Address) *maskRequest {
	r := &maskRequest{
		userID:	u.Id,
		dest: 	dest,
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
	switch buffer[0] {
	case network.AddrTypeIPv4:
		nBytes, err = decryptReader.Read(buffer[1:5])
		if err != nil {
			return
		}
		if nBytes != 4 {
			err = fmt.Errorf("Unable read complete ip, want 4, get %d", nBytes)
			return
		}
		request.dest = network.NewIPAddress(buffer[1:5], port)
	case network.AddrTypeIPv6:
		nBytes, err = decryptReader.Read(buffer[1:17])
		if err != nil {
			return
		}
		if nBytes != 16 {
			err = fmt.Errorf("Unable read complete ip, want 16, get %d", nBytes)
			return
		}
		request.dest = network.NewIPAddress(buffer[1:17], port)
	case network.AddrTypeDomain:
		_, err = decryptReader.Read(buffer[1:2])
		if err != nil {
			return
		}

		domainLen := int(buffer[1])
		nBytes, err = decryptReader.Read(buffer[2:2 + domainLen])
		if err != nil {
			return
		}
		if nBytes != domainLen {
			err = fmt.Errorf("Unable read complete domain, want %d, get %d", domainLen, nBytes)
			return
		}
		request.dest = network.NewDomainAddress(string(buffer[2:2 + domainLen]), port)
	default:
		err = fmt.Errorf("Unsupported address type: %v", buffer[0])
		return
	}

	if request.dest.Type == network.AddrTypeErr {
		err = fmt.Errorf("Illegal address: %v", request.dest)
		return
	}

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
	portBuffer := make([]byte, 2)
	binary.BigEndian.PutUint16(portBuffer, r.dest.Port)
	buffer = append(buffer, portBuffer...)

	switch r.dest.Type {
	case network.AddrTypeIPv4:
		buffer = append(buffer, network.AddrTypeIPv4)
		buffer = append(buffer, r.dest.IP...)
	case network.AddrTypeIPv6:
		buffer = append(buffer, network.AddrTypeIPv6)
		buffer = append(buffer, r.dest.IP...)
	case network.AddrTypeDomain:
		buffer = append(buffer, network.AddrTypeDomain)
		buffer = append(buffer, byte(len(r.dest.Domain)))
		buffer = append(buffer, []byte(r.dest.Domain)...)
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