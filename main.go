package main

import "github.com/go-sharp/go-runner/log"

func main() {
	log.Errorln("Hello")
	log.Info("Hello", "\n")
	log.Warnf("Hello %v \n", 43)
}
