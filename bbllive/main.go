// mss project main.go
package main

import (
	"bbllive/rtmp"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
)

var pprof bool
var listen string

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU() * 4)
	flag.StringVar(&listen, "l", ":1935", "-l=:1935")
	flag.BoolVar(&pprof, "pprof", false, "-pprof=true")
}
func main() {
	flag.Parse()
	err := rtmp.ListenAndServe(listen)
	if err != nil {
		panic(err)
	}
	log.Println("Babylon Live Server Listen At " + listen)
	if pprof {
		go func() {
			log.Println(http.ListenAndServe(":6060", nil))
		}()
	}
	select {}
}
