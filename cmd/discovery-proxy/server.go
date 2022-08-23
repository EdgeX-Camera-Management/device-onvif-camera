// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2022 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	wsdiscovery "github.com/IOTechSystems/onvif/ws-discovery"
	"github.com/google/uuid"
	"golang.org/x/net/ipv4"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	bufSize         = 8192
	channelSize     = 1000
	readTimeout     = time.Second * 3
	writeTimeout    = time.Second * 3
	multicastTTL    = 2
	wsDiscoveryPort = 3702
	probeInterval   = time.Second * 10
)

var (
	// 239.255.255.250 port 3702 is the multicast address and port used by ws-discovery
	group = net.IPv4(239, 255, 255, 250)
	dest  = &net.UDPAddr{IP: group, Port: wsDiscoveryPort}

	mIdRegex = regexp.MustCompile("MessageID>uuid:([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})<")
)

type DiscoveryProxy struct {
	netConn     net.PacketConn
	ip4Conn     *ipv4.PacketConn
	ctx         context.Context
	incomingCh  chan string
	forwardAddr net.Addr
}

func NewDiscoveryProxy() *DiscoveryProxy {
	return &DiscoveryProxy{
		incomingCh: make(chan string, channelSize),
	}
}

// Run runs the server and blocks until the context is closed
func (dp *DiscoveryProxy) Run(ctx context.Context) error {
	dp.ctx = ctx
	if err := dp.setupListener(); err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	wg.Add(3) // listen + probe + incoming loops
	go func() {
		defer wg.Done()
		dp.listenLoop()
	}()
	go func() {
		defer wg.Done()
		dp.probeLoop()
	}()
	go func() {
		defer wg.Done()
		dp.processIncomingLoop()
	}()

	// wait for all tasks to be done
	wg.Wait()
	return nil
}

func (dp *DiscoveryProxy) setupListener() error {
	c, err := net.ListenPacket("udp4", fmt.Sprintf("0.0.0.0:%d", wsDiscoveryPort))
	if err != nil {
		fmt.Printf("Error listening for ws-discovery messages on port %d: %s\n", wsDiscoveryPort, err.Error())
		return err
	}

	p := ipv4.NewPacketConn(c)
	dp.netConn = c
	dp.ip4Conn = p

	// setup all network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		fmt.Printf("Error getting network interfaces: %s\n", err.Error())
	} else {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagLoopback > 0 {
				continue // skip loopback
			}
			fmt.Printf("Calling JoinGroup on interface %s\n", iface.Name)
			if err = p.JoinGroup(&iface, &net.UDPAddr{IP: group}); err != nil {
				fmt.Printf("Error calling JoinGroup for interface %s: %s\n", iface.Name, err.Error())
			}

			// todo: should it set the multicast interface?? I think ideally we would only want to select the best one
			//if err = p.SetMulticastInterface(&iface); err != nil {
			//	fmt.Printf("Error calling SetMulticastInterface for interface %q: %s\n", iface.Name, err.Error())
			//}
		}
	}

	if err = p.SetMulticastTTL(multicastTTL); err != nil {
		fmt.Printf("Error calling SetMulticastTTL: %s\n", err.Error())
	}
	if err = p.SetMulticastLoopback(false); err != nil {
		fmt.Printf("Error turning off MulticastLoopback: %s\n", err.Error())
	}

	return nil
}

func (dp *DiscoveryProxy) listenLoop() {
	b := make([]byte, bufSize)

	// keep reading from the PacketConn until the context is done
	for {
		select {
		case <-dp.ctx.Done():
			fmt.Println("Stopping listener loop")
			return
		default:
			n, cm, src, err := dp.ip4Conn.ReadFrom(b)
			if err != nil {
				// ErrDeadlineExceeded is expected once the read timeout is expired
				if !errors.Is(err, os.ErrDeadlineExceeded) {
					fmt.Printf("Unexpected error occurred while reading ws-discovery responses: %s\n", err.Error())
				} else {
					fmt.Println("Got os.ErrDeadlineExceeded")
				}
				continue
			}
			fmt.Printf("cm: %+v\n", cm)
			fmt.Printf("src: %+v\n", src)
			data := string(b[0:n])
			if strings.Contains(data, ">http://schemas.xmlsoap.org/ws/2005/04/discovery/Probe</") {
				fmt.Printf("Received ws-discovery probe! Will attempt to proxy it!\n")
				// todo: re-work this
				//dp.probe(data, src)
			}
			dp.incomingCh <- data
		}
	}
}

func (dp *DiscoveryProxy) probe() {
	fmt.Println("Sending probe...")
	id := uuid.NewString()
	probeSOAP := wsdiscovery.BuildProbeMessage(id, nil, []string{"dn:NetworkVideoTransmitter"},
		map[string]string{"dn": "http://www.onvif.org/ver10/network/wsdl"})
	_, err := dp.ip4Conn.WriteTo([]byte(probeSOAP), nil, dest)
	if err != nil {
		fmt.Printf("Error sending probe1: %s", err.Error())
	}

	probeSOAP2 := wsdiscovery.BuildProbeMessage(id, nil, nil,
		map[string]string{"dn": "http://www.onvif.org/ver10/network/wsdl"})
	_, err = dp.ip4Conn.WriteTo([]byte(probeSOAP2.String()), nil, dest)
	if err != nil {
		fmt.Printf("Error sending probe2: %s", err.Error())
	}
}

//func (dp *DiscoveryProxy) probe(data string, proxyAddr net.Addr) {
//	id := uuid.NewString()
//	fmt.Printf("Sending probe on behalf of %v... Id=%s\n", proxyAddr, id)
//	probe := mIdRegex.ReplaceAllString(data, fmt.Sprintf("MessageID>uuid:%s<", id))
//
//	dp.forwardAddr = proxyAddr
//	_, err := dp.ip4Conn.WriteTo([]byte(probe), nil, dest)
//	if err != nil {
//		fmt.Printf("Error sending probe: %s", err.Error())
//	}
//}

func (dp *DiscoveryProxy) processIncomingLoop() {
	for payload := range dp.incomingCh {
		fmt.Printf("Received payload: %s\n", payload)
		if dp.forwardAddr != nil {
			fmt.Printf("Forwarding payload to %v\n", dp.forwardAddr)
			_, err := dp.ip4Conn.WriteTo([]byte(payload), nil, dp.forwardAddr)
			if err != nil {
				fmt.Printf("Error while trying to respond with probe: %s", err.Error())
			}
		}
		// todo: forward it? cache it?

		// todo: respond to probes for Discovery Proxies
	}
	fmt.Println("process incoming loop completed")
}

func (dp *DiscoveryProxy) probeLoop() {
	ticker := time.NewTicker(probeInterval)
	for {
		select {
		case <-dp.ctx.Done():
			fmt.Println("Stopping probe loop")
			return
		case <-ticker.C:
			dp.probe()
		}
	}
}

func (dp *DiscoveryProxy) Close() {
	if dp.netConn != nil {
		err := dp.netConn.Close()
		if err != nil {
			fmt.Printf("Error closing net packet connection: %s\n", err.Error())
		}
		dp.netConn = nil
		dp.ip4Conn = nil
	}

	if dp.incomingCh != nil {
		close(dp.incomingCh)
		dp.incomingCh = nil
	}
}
