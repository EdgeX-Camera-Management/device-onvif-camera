package main

import (
	"context"
	device_camera "github.com/edgexfoundry/device-onvif-camera"
	"github.com/edgexfoundry/device-onvif-camera/internal/driver"
	"github.com/edgexfoundry/device-onvif-camera/internal/netscan"
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/startup"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"os"
	"strings"
	"time"
)

func main() {
	ctx := context.Background()
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(),
		time.Duration(300)*time.Second)
	defer cancel()

	d := &driver.Driver{}
	os.Setenv("EDGEX_SECURITY_SECRET_STORE", "false")
	os.Setenv("WRITABLE_LOGLEVEL", "TRACE")
	os.Setenv("DEVICE_PROFILESDIR", "cmd/res/profiles")
	os.Setenv("DEVICE_DEVICESDIR", "cmd/res/devices")
	os.Args = []string{"discover", "-c", "cmd/res"}
	go startup.Bootstrap("discover-test", device_camera.Version, d)
	lc := logger.NewClient("discover-test", "TRACE")

	params := netscan.Params{
		// split the comma separated string here to avoid issues with EdgeX's Consul implementation
		Subnets:         strings.Split("10.0.0.0/24", ","),
		AsyncLimit:      4000,
		Timeout:         time.Duration(2000) * time.Millisecond,
		ScanPorts:       []string{"3702"},
		Logger:          lc,
		NetworkProtocol: netscan.NetworkUDP,
	}

	t0 := time.Now()
	result := netscan.AutoDiscover(ctx, driver.NewOnvifProtocolDiscovery(d), params)
	if ctx.Err() != nil {
		lc.Warnf("Discover process has been cancelled!", "ctxErr", ctx.Err())
	}

	lc.Debugf("NetScan result: %+v", result)
	lc.Infof("Discovered %d device(s) in %v via netscan.", len(result), time.Since(t0))
}
