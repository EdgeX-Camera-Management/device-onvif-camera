package main

import (
	"errors"
	"fmt"
	"github.com/IOTechSystems/onvif"
	wsdiscovery "github.com/IOTechSystems/onvif/ws-discovery"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/google/uuid"
	"golang.org/x/net/ipv4"
	"net"
	"os"
	"time"
)

const (
	wsDiscoveryPort = 3702
	multicastTTL    = 2
	bufSize         = 8192
	readTimeout     = time.Second * 3
)

var (
	// 239.255.255.250 port 3702 is the multicast address and port used by ws-discovery
	group = net.IPv4(239, 255, 255, 250)
	dest  = &net.UDPAddr{IP: group, Port: wsDiscoveryPort}
)

type MulticastDiscovery struct {
	netConn net.PacketConn
	ip4Conn *ipv4.PacketConn
	lc      logger.LoggingClient
}

func NewMulticastDiscovery() *MulticastDiscovery {
	return &MulticastDiscovery{
		lc: logger.NewClient("multicast", "DEBUG"),
	}
}

type MulticastResponse struct {
	src     net.Addr
	payload string
}

func getResponses(resp []MulticastResponse) []string {
	raw := make([]string, 0, len(resp))
	for _, r := range resp {
		raw = append(raw, r.payload)
	}
	return raw
}

func (md *MulticastDiscovery) Run() ([]onvif.Device, error) {
	if err := md.setupListener(); err != nil {
		return nil, err
	}
	defer md.close()
	md.probe()
	res := md.listen()
	devices, err := wsdiscovery.DevicesFromProbeResponses(getResponses(res))
	return devices, err
}

func (md *MulticastDiscovery) setupListener() error {
	c, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		md.lc.Errorf("Error listening for ws-discovery messages: %s\n", err.Error())
		return err
	}

	p := ipv4.NewPacketConn(c)
	md.netConn = c
	md.ip4Conn = p

	// setup all network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		md.lc.Errorf("Error getting network interfaces: %s", err.Error())
	} else {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagLoopback > 0 || iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagPointToPoint > 0 ||
				virtualRegex.MatchString(iface.Name) {
				// skip loopback, interfaces that are not up, and point-to-point networks
				// and certain virtual networks
				continue
			}
			md.lc.Debugf("Calling JoinGroup on interface %s", iface.Name)
			if err = p.JoinGroup(&iface, &net.UDPAddr{IP: group}); err != nil {
				md.lc.Warnf("Error calling JoinGroup for interface %s: %s", iface.Name, err.Error())
			}

			// todo: should it set the multicast interface?? I think ideally we would only want to select the best one
			//if err = p.SetMulticastInterface(&iface); err != nil {
			//	fmt.Printf("Error calling SetMulticastInterface for interface %q: %s\n", iface.Name, err.Error())
			//}
		}
	}

	if err = p.SetMulticastTTL(multicastTTL); err != nil {
		md.lc.Errorf("Error calling SetMulticastTTL: %s", err.Error())
	}
	if err = p.SetMulticastLoopback(false); err != nil {
		md.lc.Errorf("Error turning off MulticastLoopback: %s", err.Error())
	}
	if err = p.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		md.lc.Errorf("Error setting Read Deadline: %s", err.Error())
	}

	return nil
}

func (md *MulticastDiscovery) listen() []MulticastResponse {
	b := make([]byte, bufSize)
	var responses []MulticastResponse

	// keep reading from the PacketConn until the read deadline expires or an error occurs
	for {
		n, _, src, err := md.ip4Conn.ReadFrom(b)
		if err != nil {
			// ErrDeadlineExceeded is expected once the read timeout is expired
			if !errors.Is(err, os.ErrDeadlineExceeded) {
				md.lc.Errorf("Unexpected error occurred while reading ws-discovery responses: %s", err.Error())
			}
			break
		}
		response := MulticastResponse{
			src:     src,
			payload: string(b[0:n]),
		}

		md.lc.Infof("Got response from %s: %s", response.src, response.payload)

		responses = append(responses, response)
	}
	return responses
}

func (md *MulticastDiscovery) close() {
	if md.netConn != nil {
		err := md.netConn.Close()
		if err != nil {
			md.lc.Errorf("Error closing net packet connection: %s", err.Error())
		}
		md.netConn = nil
		md.ip4Conn = nil
	}
}

func (md *MulticastDiscovery) probe() {
	fmt.Println("Sending probe...")
	id := uuid.NewString()
	probeSOAP := wsdiscovery.BuildProbeMessage(id, nil, []string{"dn:NetworkVideoTransmitter"},
		map[string]string{"dn": "http://www.onvif.org/ver10/network/wsdl"})
	_, err := md.ip4Conn.WriteTo([]byte(probeSOAP), nil, dest)
	if err != nil {
		md.lc.Errorf("Error sending probe1: %s", err.Error())
	}

	probeSOAP2 := wsdiscovery.BuildProbeMessage(id, nil, nil,
		map[string]string{"dn": "http://www.onvif.org/ver10/network/wsdl"})
	_, err = md.ip4Conn.WriteTo([]byte(probeSOAP2.String()), nil, dest)
	if err != nil {
		md.lc.Errorf("Error sending probe2: %s", err.Error())
	}
}
