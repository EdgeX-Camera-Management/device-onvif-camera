// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2022 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	proxy := NewDiscoveryProxy()

	var resultErr error

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := proxy.Run(ctx)
		fmt.Printf("Proxy run returned %v\n", err)
		resultErr = err
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Waiting for os signals")
	s := <-signals
	fmt.Printf("Received %s signal. Cancelling server context.\n", s.String())
	cancel()
	proxy.Close()

	fmt.Println("Waiting for server to close")
	wg.Wait()
	fmt.Println("Server closed")
	if resultErr != nil {
		os.Exit(1)
	}
}
