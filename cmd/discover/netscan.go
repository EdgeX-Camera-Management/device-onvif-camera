package main

import (
	"context"
	"fmt"
	"github.com/edgexfoundry/device-onvif-camera/internal/driver"
	"github.com/edgexfoundry/device-onvif-camera/internal/netscan"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"net"
	"regexp"
	"strconv"
	"time"
)

var (
	// virtualRegex matches command interface names of virtual interfaces
	virtualRegex = regexp.MustCompile("(br-[a-z0-9]{12}|vmnet[0-9]+|virbr[0-9]+|vnet[0-9]+|veth[0-9a-f]{7}|docker[0-9]+)")
)

type NetScanDiscovery struct {
	params netscan.Params
	lc     logger.LoggingClient
	d      *driver.Driver
}

func NewNetScanDiscovery() *NetScanDiscovery {
	lc := logger.NewClient("netscan", "INFO")
	return &NetScanDiscovery{
		lc: lc,
		d:  &driver.Driver{},
		params: netscan.Params{
			Subnets:         getSubnets(lc),
			AsyncLimit:      4000,
			Timeout:         time.Duration(2000) * time.Millisecond,
			ScanPorts:       []string{strconv.Itoa(wsDiscoveryPort)},
			Logger:          lc,
			NetworkProtocol: netscan.NetworkUDP,
		},
	}
}

func (nd *NetScanDiscovery) Run() []netscan.ProbeResult {
	ctx := context.Background()
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(),
		time.Duration(300)*time.Second)
	defer cancel()

	proto := driver.NewOnvifProtocolDiscovery(nd.d)
	resultCh := netscan.ExecuteProbes(ctx, proto, nd.params)
	result := nd.processResultChannel(resultCh)
	if ctx.Err() != nil {
		nd.lc.Warnf("Discover process has been cancelled!", "ctxErr", ctx.Err())
	}
	return result
}

func getSubnets(lc logger.LoggingClient) []string {
	var subnets []string
	ifaces, err := net.Interfaces()
	if err != nil {
		fmt.Printf("Error getting network interfaces: %s\n", err.Error())
	} else {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagLoopback > 0 || iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagPointToPoint > 0 ||
				virtualRegex.MatchString(iface.Name) {
				// skip loopback, interfaces that are not up, and point-to-point networks
				// and certain virtual networks
				continue
			}
			addrs, err := iface.Addrs()
			if err != nil {
				fmt.Println(err.Error())
				// skip interfaces without addresses
				continue
			}
			for _, addr := range addrs {
				ipnet, ok := addr.(*net.IPNet)
				if !ok {
					continue
				}
				if ipnet.IP.To4() == nil {
					// skip non ipv4 addresses
					continue
				}
				maskSize, _ := ipnet.Mask.Size()
				if maskSize < 24 {
					// replace the netmask with /24 to speed up scan
					ipnet.Mask = net.CIDRMask(24, 32)
				}
				lc.Infof("Scanning subnet: %s for interface: %s", ipnet.String(), iface.Name)
				subnets = append(subnets, ipnet.String())
			}
		}
	}
	return subnets
}

// processResultChannel reads all incoming results until the resultCh is closed.
// it determines if a device is new or existing, and proceeds accordingly.
//
// Does not check for context cancellation because we still want to
// process any in-flight results.
func (nd *NetScanDiscovery) processResultChannel(resultCh chan []netscan.ProbeResult) []netscan.ProbeResult {
	results := make([]netscan.ProbeResult, 0)
	for probeResults := range resultCh {
		if len(probeResults) == 0 {
			continue
		}

		for _, probeResult := range probeResults {
			nd.params.Logger.Infof("Discovered: %+v", probeResult)
			results = append(results, probeResult)
		}
	}
	return results
}
