package core

import (
	"io"
	"time"

	"../log"
)

const (
	channelSize = 100
	bufferSize  = 1024 * 4
)

const (
	Active = true
	Closed = false
)

var (
	timeoutSec = 120 * time.Second
)

type FullDuplexChannel struct {
	ForwardChannel  HalfDuplexChannel
	BackwardChannel HalfDuplexChannel
}

type HalfDuplexChannel interface {
	Pop() ([]byte, bool)
	Push([]byte)
	Input(io.Reader, chan<- bool)
	Output(io.Writer, chan<- bool)
	State() bool
	Close()
}

func NewFullDuplexChannel() FullDuplexChannel {
	return FullDuplexChannel{
		ForwardChannel:  newTimedHalfDuplexChannel(channelSize, timeoutSec),
		BackwardChannel: newTimedHalfDuplexChannel(channelSize, timeoutSec),
	}
}

// timedHalfDuplexChannel implement interface HalfDuplexChannel
type timedHalfDuplexChannel struct {
	data       chan []byte
	timeoutSec time.Duration
	state      bool
}

func newTimedHalfDuplexChannel(channelSize int, timeoutSec time.Duration) *timedHalfDuplexChannel {
	return &timedHalfDuplexChannel{
		data:       make(chan []byte, channelSize),
		timeoutSec: timeoutSec,
		state:      Active,
	}
}

func (ch *timedHalfDuplexChannel) Pop() ([]byte, bool) {
	select {
	case data, ok := <-ch.data:
		return data, ok
	case <-time.After(ch.timeoutSec):
		return nil, Closed
	}
}

func (ch *timedHalfDuplexChannel) Push(data []byte) {
	if ch.state == Active {
		go func() {
			ch.data <- data
		}()
	}
}

// data flow: reader -> channel
func (ch *timedHalfDuplexChannel) Input(reader io.Reader, finish chan<- bool) {
	defer ch.Close()

	reader = NewTimedReader(reader, ch.timeoutSec)
	for ch.state == Active {
		buffer := make([]byte, bufferSize)
		nBytes, err := reader.Read(buffer)
		if nBytes > 0 {
			ch.data <- buffer[:nBytes]
		}
		if err == io.EOF {
			break
		} else if err != nil {
			log.Warning("Err in channel Input(): %v", err)
			finish <- false
			return
		}
	}
	finish <- true
}

// data flow: channel -> writer
func (ch *timedHalfDuplexChannel) Output(writer io.Writer, finish chan<- bool) {
	writer = NewTimedWriter(writer, ch.timeoutSec)
	for buf := range ch.data {
		_, err := writer.Write(buf)
		if err != nil {
			log.Warning("Err in channel Output(): %v", err)
			finish <- false
			return
		}
	}
	finish <- true
}

func (ch *timedHalfDuplexChannel) Close() {
	if ch.state == Active {
		ch.state = Closed
		<-time.After(ch.timeoutSec + 5*time.Second)

		defer func() {
			recover()
		}()
		close(ch.data)
	}
}

func (ch *timedHalfDuplexChannel) State() bool {
	return ch.state
}
