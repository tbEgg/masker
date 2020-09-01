package network

import (
	"io"
)

func CloseConnection(closer io.Closer, readFinish <-chan bool, writeFinish <-chan bool) {
	<-readFinish
	<-writeFinish
	closer.Close()
}
