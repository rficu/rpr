package main

import (
	"github.com/rficu/rpr/pkg"
	"math/rand"
	"time"
)

func main() {

	rand.Seed(time.Now().UnixNano())

	go rpr.InitBootstrap("127.0.0.1:2222")
	time.Sleep(500 * time.Millisecond)

	go rpr.InitNode("127.0.0.1:2222", 8100, 5, "COMPAT")
	time.Sleep(1000 * time.Millisecond)

	go rpr.InitNode("127.0.0.1:2222", 8200, 10, "COMPAT")
	time.Sleep(5 * 1000 * time.Millisecond)

	go rpr.InitNode("127.0.0.1:2222", 8300, 1, "COMPAT")

	for {
		time.Sleep(5 * 1000 * time.Millisecond)
	}
	println("end...\n")
}
