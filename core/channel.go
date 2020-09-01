package core

import (
	"io"
	// "../log"
)

const (
	channelSize = 100
	bufferSize  = 1024 * 32
)

type FullDuplexChannel struct {
	ForwardChannel  HalfDuplexChannel
	BackwardChannel HalfDuplexChannel
}

type HalfDuplexChannel chan []byte

func NewFullDuplexChannel() FullDuplexChannel {
	return FullDuplexChannel{
		ForwardChannel:  make(HalfDuplexChannel, channelSize),
		BackwardChannel: make(HalfDuplexChannel, channelSize),
	}
}

func (ch HalfDuplexChannel) Pop() ([]byte, bool) {
	data, ok := <-ch
	return data, ok
}

func (ch HalfDuplexChannel) Push(data []byte) {
	go func() {
		ch <- data
	}()
}

// data flow: reader -> channel
func (ch HalfDuplexChannel) Input(reader io.Reader, finish chan<- bool) {
	for {
		buffer := make([]byte, bufferSize)
		nBytes, err := reader.Read(buffer)
		if nBytes > 0 {
			ch <- buffer[:nBytes]
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
func (ch HalfDuplexChannel) Output(writer io.Writer, finish chan<- bool) {
	for buffer := range ch {
		_, err := writer.Write(buffer)
		if err != nil {
			finish <- false
			return
		}
	}
	finish <- true
}
