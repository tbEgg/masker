package socks

import (
	"masker/core"
)

type SocksListenerConstructor struct{}

func (SocksListenerConstructor) Create(node *core.Node, configFile string) (core.Listener, error) {
	return NewSocksListener(node, configFile)
}

func init() {
	core.RegisterListenerConstructor("socks", SocksListenerConstructor{})
}
