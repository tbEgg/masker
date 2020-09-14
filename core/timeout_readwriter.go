package core

import (
	"errors"
	"io"
	"time"
)

var (
	ErrReadTimeout  = errors.New("reading time out")
	ErrWriteTimeout = errors.New("writing time out")
)

type timedReader struct {
	reader     io.Reader
	timeoutSec time.Duration
}

func NewTimedReader(reader io.Reader, timeoutSec time.Duration) io.Reader {
	return &timedReader{
		reader:     reader,
		timeoutSec: timeoutSec,
	}
}

func (reader *timedReader) Read(buf []byte) (nBytes int, err error) {
	ch := make(chan bool)
	go func() {
		nBytes, err = reader.reader.Read(buf)
		ch <- true
	}()

	select {
	case <-ch:
		return
	case <-time.After(reader.timeoutSec):
		return 0, ErrReadTimeout
	}
}

type timedWriter struct {
	writer     io.Writer
	timeoutSec time.Duration
}

func NewTimedWriter(writer io.Writer, timeoutSec time.Duration) io.Writer {
	return &timedWriter{
		writer:     writer,
		timeoutSec: timeoutSec,
	}
}

func (writer *timedWriter) Write(buf []byte) (nBytes int, err error) {
	ch := make(chan bool)
	go func() {
		nBytes, err = writer.writer.Write(buf)
		ch <- true
	}()

	select {
	case <-ch:
		return
	case <-time.After(writer.timeoutSec):
		return 0, ErrWriteTimeout
	}
}
