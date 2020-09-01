package account

import (
	"crypto/md5"
	"encoding/hex"

	"../log"
)

const (
	IDBytesLen = 16
)

type ID struct {
	Text   string
	Bytes  []byte
	cmdKey []byte
}

func NewID(id string) (*ID, error) {
	idBytes, err := UUIDToID(id)
	if err != nil {
		return nil, err
	}

	md5Hash := md5.New()
	md5Hash.Write(idBytes)
	md5Hash.Write([]byte("c48619fe-8f02-49e0-b9e9-edf763e17e21"))
	key := md5Hash.Sum(nil)

	return &ID{
		Text:   id,
		Bytes:  idBytes,
		cmdKey: key[:],
	}, nil
}

func (id *ID) CmdKey() []byte {
	return id.cmdKey
}

// copy from v2ray
var byteGroups = []int{8, 4, 4, 4, 12}

// TODO: leverage a full functional UUID library
func UUIDToID(uuid string) (v []byte, err error) {
	v = make([]byte, 16)

	text := []byte(uuid)
	if len(text) < 32 {
		err = log.Error("uuid: invalid UUID string: %s", text)
		return
	}

	b := v[:]

	for _, byteGroup := range byteGroups {
		if text[0] == '-' {
			text = text[1:]
		}

		_, err = hex.Decode(b[:byteGroup/2], text[:byteGroup])

		if err != nil {
			return
		}

		text = text[byteGroup:]
		b = b[byteGroup/2:]
	}

	return
}
