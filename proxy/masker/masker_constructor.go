package masker

import (
	"../../core"
)

type MaskCallerConstructor struct{}

func (MaskCallerConstructor) Create(configFile string) (core.Caller, error) {
	return NewMaskCaller(configFile)
}

type MaskListenerConstructor struct{}

func (MaskListenerConstructor) Create(node *core.Node, configFile string) (core.Listener, error) {
	return NewMaskListener(node, configFile)
}

func init() {
	core.RegisterCallerConstructor("mask", MaskCallerConstructor{})
	core.RegisterListenerConstructor("mask", MaskListenerConstructor{})
}