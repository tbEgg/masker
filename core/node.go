package core

import (
	"../log"
	"../network"
)

type Node struct {
	ListenEnd	Listener
	CallEnd		Caller
	Config		NodeConfig
}

type Listener interface {
	Listen(uint16) error
}

type Caller interface {
	Call(FullDuplexChannel, network.Destination) error
}

type ListenerConstructor interface {
	Create(*Node, string) (Listener, error)
}

type CallerConstructor interface {
	Create(string) (Caller, error)
}


func NewNode(config NodeConfig) (*Node, error) {
	node := new(Node)

	listenerConstructor, ok := listenerConstructorSet[config.ListenEndConfig.Protocol]
	if !ok {
		panic(log.Error("No such listener protocol: %v.", config.ListenEndConfig.Protocol))
	}
	listener, err := listenerConstructor.Create(node, config.ListenEndConfig.ConfigFile)
	if err != nil {
		return node, log.Error("can't create listener")
	}
	node.ListenEnd = listener

	callerConstructor, ok := callerConstructorSet[config.CallEndConfig.Protocol]
	if !ok {
		panic(log.Error("No such caller protocol: %v.", config.CallEndConfig.Protocol))
	}
	caller, err := callerConstructor.Create(config.CallEndConfig.ConfigFile)
	if err != nil {
		return node, log.Error("can't create caller")
	}
	node.CallEnd = caller

	node.Config = config

	return node, nil
}

var (
	listenerConstructorSet	= make(map[string]ListenerConstructor)
	callerConstructorSet	= make(map[string]CallerConstructor)
)

func RegisterListenerConstructor(protocol string, constructor ListenerConstructor) {
	listenerConstructorSet[protocol] = constructor
}

func RegisterCallerConstructor(protocol string, constructor CallerConstructor) {
	callerConstructorSet[protocol] = constructor
}

func (node *Node) Start() error {
	return node.ListenEnd.Listen(node.Config.Port)
}

func (node *Node) NewConnectionAccept(dest network.Destination) (FullDuplexChannel, error) {
	channel := NewFullDuplexChannel()
	go node.CallEnd.Call(channel, dest)
	return channel, nil
}