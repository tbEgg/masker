package masker

import (
	"net"
	"strconv"
	"crypto/md5"

	"../../log"
	"../../core"
	"../../account"
	"../../cryption"
)

type MaskListener struct {
	node	*core.Node
	userSet	account.UserSet
}

func NewMaskListener(node *core.Node, configFile string) (*MaskListener, error) {
	config, err := loadListenerConfig(configFile)
	if err != nil {
		log.Error("Err in loading mask listener config: %v.", err)
		return nil, err
	}

	userList := make([]account.User, 0)
	for _, tmpUserConfig := range config.UserList {
		if tmpUser, ok := tmpUserConfig.toUser(); ok {
			userList = append(userList, tmpUser)
		}
	}
	if len(userList) == 0 {
		return nil, log.Error("Check your config, don't find any allowed user!")
	}

	userSet, err := account.NewTimedUserSet(userList...)
	if err != nil {
		return nil, log.Error("Err in creating user set: %v", err)
	}

	return &MaskListener{
		node:		node,
		userSet:	userSet,
	}, nil
}

func (listener *MaskListener) Listen(port uint16) error {
	ln, err := net.Listen("tcp", ":" + strconv.Itoa(int(port)))
	if err != nil {
		return err
	}
	log.Info("Listening on port: %d...", port)

	go listener.acceptConnection(ln)
	return nil
}

func (listener *MaskListener) acceptConnection(ln net.Listener) {
	// set max handle connections?
	for true {
		conn, err := ln.Accept()
		if err != nil {
			log.Error("Err in accepting socket connection: %v.", err)
		} else {
			go listener.handleConnection(conn)
		}
	}
}

func (listener *MaskListener) handleConnection(conn net.Conn) error {
	defer conn.Close()

	// read request
	maskRequest, err := readMaskRequest(conn, listener.userSet)
	if err != nil {
		log.Error("Err in reading mask request: %v", err)
		return err
	}

	channel, err := listener.node.NewConnectionAccept(maskRequest.dest)
	if err != nil {
		log.Error("Err in calling destination: %v", err)
		return err
	}
	readFinish  := make(chan bool, 1)
	writeFinish := make(chan bool, 1)

	// transmit request
	decryptReader, err := cryption.NewAESDecryptReader(conn, maskRequest.requestKey[:], maskRequest.requestIV[:])
	if err != nil {
		log.Error("Err in creating decrypt reader: %v", err)
		return err
	}
	go channel.ForwardChannel.Input(decryptReader, readFinish)

	// send response
	key := md5.Sum(maskRequest.requestKey[:])
	IV := md5.Sum(maskRequest.requestIV[:])
	encryptWriter, err := cryption.NewAESEncryptWriter(conn, key[:], IV[:])
	if err != nil {
		log.Error("Err in creating encrypt writer: %v", err)
		return err
	}

	response := newMaskResponse(maskRequest)
	buffer := make([]byte, 0, 1024)
	buffer = append(buffer, response[:]...)
	if payload, ok := channel.BackwardChannel.Pop(); ok {
		buffer = append(buffer, payload...)
		encryptWriter.Write(buffer)
		go channel.BackwardChannel.Output(encryptWriter, writeFinish)
		<-writeFinish
	} else {
		log.Error("Err in first response block.")
	}
	
	return nil
}