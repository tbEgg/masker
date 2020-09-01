package log

import (
	"errors"
	"fmt"
	"log"
)

type logLevel int

const (
	DebugLevel   = logLevel(0)
	InfoLevel    = logLevel(1)
	WarningLevel = logLevel(2)
	ErrorLevel   = logLevel(3)
)

const (
	DebugStateLogFlag   = log.Ldate | log.Ltime | log.Lshortfile
	ReleaseStateLogFlag = log.LstdFlags
)

func init() {
	log.SetFlags(DebugStateLogFlag)
}

var curLogLevel = InfoLevel

func SetCurLogLevel(level logLevel) {
	curLogLevel = level

	if level == DebugLevel {
		log.SetFlags(DebugStateLogFlag)
	} else {
		log.SetFlags(ReleaseStateLogFlag)
	}
}

func Debug(format string, v ...interface{}) {
	writeLog(DebugLevel, "[Debug]", format, v...)
}

func Info(format string, v ...interface{}) {
	writeLog(InfoLevel, "[Info]", format, v...)
}

func Warning(format string, v ...interface{}) {
	writeLog(WarningLevel, "[Warning]", format, v...)
}

func Error(format string, v ...interface{}) error {
	data := writeLog(ErrorLevel, "[Error]", format, v...)
	return errors.New(data)
}

func writeLog(level logLevel, prefix, format string, v ...interface{}) string {
	if level < curLogLevel {
		return ""
	}

	data := fmt.Sprintf(format, v...)
	log.Print(prefix + data)
	return data
}
