package main

//go:generate go run directives_generate.go
//go:generate go run owners_generate.go

import (
	"fmt"
	"time"

	"github.com/coredns/coredns/coremain"

	// Plug in CoreDNS
	_ "github.com/coredns/coredns/core/plugin"
)

func main() {
	go func() {
		for {
			tt()
			time.Sleep(5 * time.Second)
		}
	}()

	coremain.Run()
}

func tt() {
	t := time.Now()
	fmt.Println(t)
}
