package core

import (
	"io"
	"time"
)

const (
	channelSize = 100
	bufferSize  = 1024 * 4
	timeoutSec  = 60e9
)

type FullDuplexChannel struct {
	ForwardChannel  HalfDuplexChannel
	BackwardChannel HalfDuplexChannel
}

type HalfDuplexChannel interface {
	Pop() ([]byte, bool)
	Push([]byte)
	Input(reader io.Reader, finish chan<- bool)
	Output(writer io.Writer, finish chan<- bool)
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
}

func newTimedHalfDuplexChannel(channelSize int, timeoutSec time.Duration) *timedHalfDuplexChannel {
	return &timedHalfDuplexChannel{
		data:       make(chan []byte, channelSize),
		timeoutSec: timeoutSec,
	}
}

func (ch *timedHalfDuplexChannel) Pop() ([]byte, bool) {
	select {
	case data, ok := <-ch.data:
		return data, ok
	case <-time.After(ch.timeoutSec):
		return nil, false
	}
}

func (ch *timedHalfDuplexChannel) Push(data []byte) {
	go func() {
		ch.data <- data
	}()
}

// data flow: reader -> channel
func (ch *timedHalfDuplexChannel) Input(reader io.Reader, finish chan<- bool) {
	buffer := make([]byte, bufferSize)
	for {
		nBytes, err := reader.Read(buffer)
		if nBytes > 0 {
			ch.data <- buffer[:nBytes]
		}
		if err == io.EOF {
			break
		} else if err != nil {
			finish <- false
			return
		}
	}
	finish <- true
}

// data flow: channel -> writer
func (ch *timedHalfDuplexChannel) Output(writer io.Writer, finish chan<- bool) {
	for {
		select {
		case buffer := <-ch.data:
			_, err := writer.Write(buffer)
			if err != nil {
				finish <- false
				return
			}
		case <-time.After(ch.timeoutSec):
			close(ch.data)
			finish <- true
			return
		}
	}
}

type Timer interface {
	SetTimeoutSec(timeoutSec time.Duration)
}

func (ch *timedHalfDuplexChannel) SetTimeoutSec(timeoutSec time.Duration) {
	ch.timeoutSec = timeoutSec
}
