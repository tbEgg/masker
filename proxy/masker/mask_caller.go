package masker

import (
	"bytes"
	"crypto/md5"
	"io"
	"math/rand"
	"net"

	"../../account"
	"../../core"
	"../../cryption"
	"../../log"
	"../../network"
)

type MaskCaller struct {
	nextNodeList []nextNode // list of nodes can be connected
}

type nextNode struct {
	destination network.Destination // node ip address
	userList    []account.User      // users that node allows to access
}

func NewMaskCaller(configFile string) (*MaskCaller, error) {
	nextNodeConfigList, err := loadCallerConfig(configFile)
	if err != nil {
		log.Error("Err in loading mask caller config: %v.", err)
		return nil, err
	}

	nextNodeList := make([]nextNode, 0, len(nextNodeConfigList))
	for _, tmpNextNodeConfig := range nextNodeConfigList {
		if tmpNextNode, ok := tmpNextNodeConfig.toNextNode(); ok {
			nextNodeList = append(nextNodeList, tmpNextNode)
		}
	}
	if len(nextNodeList) == 0 {
		return nil, log.Error("Check your config, don't find any accessible node!")
	}

	return &MaskCaller{
		nextNodeList: nextNodeList,
	}, nil
}

/**
 * Build link with next node(not the target address)
 * then encrypt data (read from channel) and transmit it
 *
 * dest: final target address
 *
 */
func (caller *MaskCaller) Call(channel core.FullDuplexChannel, dest network.Destination) error {
	nextNodeDestination, chosenUser := caller.pickNextNode()

	conn, err := net.Dial(nextNodeDestination.Network(), nextNodeDestination.String())
	if err != nil {
		log.Error("Err in opening %s connection: %v.", nextNodeDestination.Network(), err)
		return err
	}
	log.Info("Connecting to %s succeed.", nextNodeDestination.String())

	request := newMaskRequest(chosenUser, dest)

	writeFinish := make(chan bool, 1)
	go sendRequest(conn, channel.ForwardChannel, writeFinish, request)

	readFinish := make(chan bool, 1)
	go receiveResponse(conn, channel.BackwardChannel, readFinish, request)

	go network.CloseConnection(conn, readFinish, writeFinish)
	return nil
}

func (caller *MaskCaller) pickNextNode() (network.Destination, account.User) {
	nextNodeNum := len(caller.nextNodeList)
	chosenNode := caller.nextNodeList[rand.Intn(nextNodeNum)]

	userNum := len(chosenNode.userList)
	chosenUser := chosenNode.userList[rand.Intn(userNum)]

	return chosenNode.destination, chosenUser
}

// encrypt request then send to chosen next node
// read data from channel -> write data to conn
func sendRequest(writer io.Writer, channel core.HalfDuplexChannel, finish chan<- bool, request *maskRequest) (err error) {
	defer func() {
		if err != nil {
			finish <- false
		}
	}()

	encryptWriter, err := cryption.NewAESEncryptWriter(writer, request.requestKey[:], request.requestIV[:])
	if err != nil {
		log.Error("Err in creating encrypt writer: %v", err)
		return
	}

	// send first packet of payload together with request, in favor of small request
	if payload, ok := channel.Pop(); ok {
		encryptedRequest, err := request.encryptedByteSlice()
		if err != nil {
			log.Error("Err in serializing request: %v", err)
			return err
		}
		encryptWriter.Encrypt(payload)

		firstPacket := append(encryptedRequest, payload...)
		_, err = writer.Write(firstPacket)
		if err != nil {
			log.Error("Err in send first packet: %v", err)
			return err
		}

		// than send other
		go channel.Output(encryptWriter, finish)
	} else {
		err = log.Error("Can't read first block.")
	}
	return
}

// decrypt response and send back
// read data from conn -> write data to channel
func receiveResponse(reader io.Reader, channel core.HalfDuplexChannel, finish chan<- bool, request *maskRequest) (err error) {
	defer func() {
		if err != nil {
			finish <- false
		}
	}()

	key := md5.Sum(request.requestKey[:])
	IV := md5.Sum(request.requestIV[:])
	decryptReader, err := cryption.NewAESDecryptReader(reader, key[:], IV[:])
	if err != nil {
		log.Error("Err in creating decrypt reader: %v", err)
		return
	}

	// check response
	response := maskResponse{}
	_, err = decryptReader.Read(response[:])
	if err != nil {
		log.Error("Err in reading mask response: %v", err)
		return
	}
	if !bytes.Equal(response[:], request.responseHeader[:]) {
		err = log.Error("Unexpected response header.")
		return
	}

	go channel.Input(decryptReader, finish)
	return
}
