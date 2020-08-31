package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"
)

func handleTime(w http.ResponseWriter, r *http.Request) {
	log.Println("calling time api")
	fmt.Fprintf(w, "current time is: %v", time.Now())
}

func handleOSInfo(w http.ResponseWriter, r *http.Request) {
	log.Println("calling os info api")
	fmt.Fprintf(w, "server os %v, browser info %v", runtime.GOOS, r.UserAgent())
}

func main() {
	log.Println("starting http server on 127.0.0.1:8080")
	http.HandleFunc("/time", handleTime)
	http.HandleFunc("/info", handleOSInfo)
	log.Println(http.ListenAndServe(":8080", nil))

}
