package socks

import (
	"net"
	"strconv"

	"masker/core"
	"masker/log"
)

type SocksListener struct {
	node   *core.Node
	config socksConfig
}

func NewSocksListener(node *core.Node, configFile string) (*SocksListener, error) {
	config, err := loadConfig(configFile)
	if err != nil {
		return nil, err
	}
	return &SocksListener{
		node:   node,
		config: config,
	}, nil
}

func (listener *SocksListener) Listen(port uint16) error {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(int(port)))
	if err != nil {
		return err
	}
	log.Info("Listening on port: %d...", port)

	go listener.acceptConnection(ln)
	return nil
}

func (listener *SocksListener) acceptConnection(ln net.Listener) {
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

func (listener *SocksListener) handleConnection(conn net.Conn) error {
	defer conn.Close()
	log.Debug("Handling a new connection.")

	// client request to choose auth method
	authRequest, err := readAuthentication(conn)
	if err != nil {
		log.Error("Err in reading auth: %v.", err)
		return err
	}
	log.Debug("auth request: %v", authRequest)

	// server choose an appropriate method and reply
	authMethod := listener.config.authMethod
	if authRequest.hasSupportedMethod(authMethod) == false {
		authResponse := newAuthenticationResponse(authNoAcceptableMethod)
		err = writeResponse(conn, authResponse)
		if err != nil {
			log.Error("Err in writing response to auth request: %v.", err)
		}
		return log.Error("Server don't support any methods that client have.")
	} else {
		authResponse := newAuthenticationResponse(authMethod)
		err = writeResponse(conn, authResponse)
		if err != nil {
			log.Error("Err in writing response to auth request: %v.", err)
			return err
		}
		log.Debug("auth response: %v", authResponse)
	}

	if authMethod == authUserPass {
		// additional part, verify the user
		userpassRequest, err := readUserPass(conn)
		if err != nil {
			log.Error("Err in reading username and password: %v", err)
			return err
		}
		log.Debug("user pass request: %v", userpassRequest)

		status := userpassRequest.verifyUser()
		userpassResponse := newUserPassResponse(status)
		err = writeResponse(conn, userpassResponse)
		if err != nil {
			log.Error("Err in responsing to verify user: %v.", err)
			return err
		}
		if status != validUser {
			return log.Error("Invalid user.")
		}
		log.Debug("user pass response: %v", userpassResponse)
	}

	// client show the destination address
	destRequest, err := readDestination(conn)
	if err != nil {
		log.Error("Err in reading the destination address: %v.", err)
		return err
	}
	log.Debug("final request: %v", destRequest)

	// server reply
	destResponse := newConfirmDestinationResponse(destRequest)
	if destRequest.command != cmdConnect {
		destResponse.statusCode = statusCommandNotSupported
		err = writeResponse(conn, destResponse)
		if err != nil {
			log.Error("Err in confirming the destination: %v", err)
		}
		return log.Error("Unsupported socks command %d", destRequest.command)
	} else {
		err = writeResponse(conn, destResponse)
		if err != nil {
			log.Error("Err in confirming the destination: %v.", err)
			return err
		}
		log.Debug("final response: %v", destResponse)
	}

	// start communicating with caller
	dest, err := destResponse.Destination()
	if err != nil {
		log.Error("Err in getting the destination: %v.", err)
		return err
	}
	log.Debug("Destination is :%v", dest)

	channel, err := listener.node.NewConnectionAccept(dest)
	if err != nil {
		log.Error("Err in calling destination: %v.", err)
		return err
	}

	readFinish := make(chan bool, 1)
	go channel.ForwardChannel.Input(conn, readFinish)

	writeFinish := make(chan bool, 1)
	go channel.BackwardChannel.Output(conn, writeFinish)

	<-writeFinish
	log.Debug("Connection Finished.")
	return nil
}
