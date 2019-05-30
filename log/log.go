package log

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/fatih/color"
)

var red = color.New(color.FgRed)
var green = color.New(color.FgGreen)
var yellow = color.New(color.FgYellow)

type Level int

const (
	InfoLevel = iota
	WarnLevel
	ErrorLevel
)

type writer func(p []byte) (n int, err error)

func (w writer) Write(p []byte) (n int, err error) {
	return w(p)
}

func CreateWriter(lvl Level, prefix string) io.Writer {
	return writer(func(p []byte) (n int, err error) {
		log(lvl, "[%v] %v", prefix, string(p))
		return len(p), nil
	})
}

func Info(args ...interface{}) {
	log(InfoLevel, "", args...)
}

func Infoln(args ...interface{}) {
	log(InfoLevel, "", append(args, "\n")...)
}

func Infof(format string, args ...interface{}) {
	log(InfoLevel, format, args...)
}

func Warn(args ...interface{}) {
	log(WarnLevel, "", args...)
}

func Warnln(args ...interface{}) {
	log(WarnLevel, "", append(args, "\n")...)
}

func Warnf(format string, args ...interface{}) {
	log(WarnLevel, format, args...)
}

func Error(args ...interface{}) {
	log(ErrorLevel, "", args...)
}

func Errorln(args ...interface{}) {
	log(ErrorLevel, "", append(args, "\n")...)
}

func Errorf(format string, args ...interface{}) {
	log(ErrorLevel, format, args...)
}

func log(level Level, format string, args ...interface{}) {
	date := time.Now().Format("2006-01-02 15:04:05")
	switch level {
	case WarnLevel:
		yellow.Printf("%v|warn: ", date)
	case ErrorLevel:
		red.Printf("%v|erro: ", date)
	default:
		green.Printf("%v|info: ", date)
	}

	if format == "" {
		fmt.Fprint(os.Stdout, args...)
	} else {
		fmt.Fprintf(os.Stdout, format, args...)
	}
}
