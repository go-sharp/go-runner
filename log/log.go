package log

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
)

var red = color.New(color.FgRed)
var green = color.New(color.FgGreen)
var yellow = color.New(color.FgYellow)

const (
	info = iota
	warn
	err
)

func Info(args ...interface{}) {
	log(info, "", args...)
}

func Infoln(args ...interface{}) {
	log(info, "", append(args, "\n")...)
}

func Infof(format string, args ...interface{}) {
	log(info, format, args...)
}

func Warn(args ...interface{}) {
	log(warn, "", args...)
}

func Warnln(args ...interface{}) {
	log(warn, "", append(args, "\n")...)
}

func Warnf(format string, args ...interface{}) {
	log(warn, format, args...)
}

func Error(args ...interface{}) {
	log(err, "", args...)
}

func Errorln(args ...interface{}) {
	log(err, "", append(args, "\n")...)
}

func Errorf(format string, args ...interface{}) {
	log(err, format, args...)
}

func log(level int, format string, args ...interface{}) {
	date := time.Now().Format("2006-01-02 15:04:05")
	switch level {
	case warn:
		yellow.Printf("%v|warn: ", date)
	case err:
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
