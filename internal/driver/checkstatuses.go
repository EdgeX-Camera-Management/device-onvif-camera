// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2022 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"net"
	"sync"
	"time"

	"github.com/IOTechSystems/onvif"
	sdkModel "github.com/edgexfoundry/go-mod-core-contracts/v2/models"
)

// checkStatuses loops through all registered devices and tries to determine the most accurate connection state
func (d *Driver) checkStatuses() {
	d.lc.Debug("checkStatuses has been called")
	wg := sync.WaitGroup{}
	for _, device := range d.sdkService.Devices() {
		device := device                        // save the device value within the closure
		if device.Name == d.sdkService.Name() { // skip control plane device
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			status := d.testConnectionMethods(device)

			if err := d.updateDeviceStatus(device.Name, status); err != nil {
				d.lc.Warnf("Could not update device status for device %s: %s", device.Name, err.Error())
			}
		}()
	}
	wg.Wait()
}

// testConnectionMethods will try to determine the state using different device calls
// and return the most accurate status
// Higher degrees of connection are tested first, becuase if they
// succeed, the lower levels of connection will too
func (d *Driver) testConnectionMethods(device sdkModel.Device) (status string) {

	// sends get capabilities command to device (does not require credentials)
	devClient, edgexErr := d.newTemporaryOnvifClient(device)
	if edgexErr != nil {
		d.lc.Debugf("Connection to %s failed when creating client: %s", device.Name, edgexErr.Message())
		// onvif connection failed, so lets probe it
		if d.tcpProbe(device) {
			return Reachable
		}
		return Unreachable

	}

	// sends get device information command to device (requires credentials)
	_, edgexErr = devClient.callOnvifFunction(onvif.DeviceWebService, onvif.GetDeviceInformation, []byte{})
	if edgexErr != nil {
		d.lc.Debugf("%s command failed for device %s when using authentication: %s", onvif.GetDeviceInformation, device.Name, edgexErr.Message())
		return UpWithoutAuth
	}

	return UpWithAuth
}

// tcpProbe attempts to make a connection to a specific ip and port list to determine
// if there is a service listening at that ip+port.
func (d *Driver) tcpProbe(device sdkModel.Device) bool {
	proto, ok := device.Protocols[OnvifProtocol]
	if !ok {
		d.lc.Warnf("Device %s is missing required %s protocol info, cannot send probe.", device.Name, OnvifProtocol)
		return false
	}
	addr := proto[Address]
	port := proto[Port]

	if addr == "" || port == "" {
		d.lc.Warnf("Device %s has no network address, cannot send probe.", device.Name)
		return false
	}
	host := addr + ":" + port

	conn, err := net.DialTimeout("tcp", host, time.Duration(d.config.AppCustom.ProbeTimeoutMillis)*time.Millisecond)
	if err != nil {
		d.lc.Debugf("Connection to %s failed when using simple tcp dial, Error: %s ", device.Name, err.Error())
		return false
	}
	defer conn.Close()
	return true
}

func (d *Driver) updateDeviceStatus(deviceName string, status string) error {
	return d.sdkService.UpdateDeviceWithLock(deviceName, func(device *sdkModel.Device) bool {
		// todo: maybe have connection levels known as ints, so that way we can log at different levels based on
		//       if the connection level went up or down
		shouldUpdate := false

		oldStatus := device.Protocols[OnvifProtocol][DeviceStatus]
		if oldStatus != status {
			d.lc.Infof("Device status for %s is now %s (used to be %s)", device.Name, status, oldStatus)
			device.Protocols[OnvifProtocol][DeviceStatus] = status
			shouldUpdate = true
		}

		if status != Unreachable {
			device.Protocols[OnvifProtocol][LastSeen] = time.Now().Format(time.UnixDate)
			shouldUpdate = true
		}

		return shouldUpdate
	})
}

// taskLoop manages all of our custom background tasks such as checking camera statuses at regular intervals
func (d *Driver) taskLoop() {
	d.configMu.RLock()
	interval := d.config.AppCustom.CheckStatusInterval
	d.configMu.RUnlock()
	if interval > maxStatusInterval { // check the interval
		d.lc.Warnf("Status interval of %d seconds is larger than the maximum value of %d seconds. Status interval has been set to the max value.", interval, maxStatusInterval)
		interval = maxStatusInterval
	}

	statusTicker := time.NewTicker(time.Duration(interval) * time.Second) // TODO: Support dynamic updates for ticker interval

	defer statusTicker.Stop()

	d.lc.Info("Starting task loop.")

	for {
		select {
		case <-d.taskCh:
			return
		case <-statusTicker.C:
			start := time.Now()
			d.checkStatuses() // checks the status of every device
			d.lc.Debugf("checkStatuses completed in: %v", time.Since(start))
		}
	}
}
