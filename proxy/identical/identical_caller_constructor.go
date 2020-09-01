package identical

import (
	"../../core"
)

type IdenticalCallerConstructor struct{}

func (IdenticalCallerConstructor) Create(configFile string) (core.Caller, error) {
	return NewIdenticalCaller(configFile)
}

func init() {
	core.RegisterCallerConstructor("identical", IdenticalCallerConstructor{})
}
