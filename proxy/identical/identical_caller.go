package identical

import (
	"net"

	"../../log"
	"../../core"
	"../../network"
)

type IdenticalCaller struct {
	configFile string
}

func NewIdenticalCaller(configFile string) (*IdenticalCaller, error) {
	return &IdenticalCaller{
		configFile: configFile,
	}, nil
}

func (caller *IdenticalCaller) Call(channel core.FullDuplexChannel, dest network.Destination) error {
	conn, err := net.Dial(dest.Network(), dest.String())
	if err != nil {
		log.Error("Err in opening %s connection: %v.", dest.Network(), err)
		return err
	}
	log.Info("Connecting to %s succeed.", dest.String())

	// read request from channel and write in conn
	writeFinish := make(chan bool, 1)
	go channel.ForwardChannel.Output(conn, writeFinish)

	// read response from conn and write in channel
	readFinish := make(chan bool, 1)
	go channel.BackwardChannel.Input(conn, readFinish)

	go network.CloseConnection(conn, readFinish, writeFinish)
	return nil
}