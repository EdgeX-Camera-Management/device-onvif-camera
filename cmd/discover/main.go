package main

import (
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"time"
)

func main() {
	lc := logger.NewClient("discover-test", "DEBUG")

	t0 := time.Now()
	md := NewMulticastDiscovery()
	responses, err := md.Run()
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
	}
	lc.Infof("Discovered %d device(s) in %v via multicast.", len(responses), time.Since(t0))

	t1 := time.Now()
	nd := NewNetScanDiscovery()
	result := nd.Run()
	lc.Infof("Discovered %d device(s) in %v via netscan.", len(result), time.Since(t1))
}
