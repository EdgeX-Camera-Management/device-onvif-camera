// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2022 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"testing"

	sdkModel "github.com/edgexfoundry/device-sdk-go/v2/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/models"
	contract "github.com/edgexfoundry/go-mod-core-contracts/v2/models"
	"github.com/stretchr/testify/assert"
)

func createTestDeviceList() []contract.Device {
	return []models.Device{
		{
			Name: "device-onvif-camera", Protocols: map[string]models.ProtocolProperties{
				OnvifProtocol: map[string]string{
					EndpointRefAddress: "abc",
				},
			},
		},
		{
			Name: "testDevice1", Protocols: map[string]models.ProtocolProperties{
				OnvifProtocol: map[string]string{
					EndpointRefAddress: "123",
				},
			},
		},
		{
			Name: "testDevice2", Protocols: map[string]models.ProtocolProperties{
				OnvifProtocol: map[string]string{
					EndpointRefAddress: "456",
				},
			},
		},
		{
			Name: "testDevice3", Protocols: map[string]models.ProtocolProperties{
				OnvifProtocol: map[string]string{
					EndpointRefAddress: "789",
				},
			},
		},
	}
}

func createDiscoveredList() []sdkModel.DiscoveredDevice {
	return []sdkModel.DiscoveredDevice{
		{
			Name: "testDevice1", Protocols: map[string]models.ProtocolProperties{
				OnvifProtocol: map[string]string{
					EndpointRefAddress: "123",
				},
			},
		},
		{
			Name: "testDevice2", Protocols: map[string]models.ProtocolProperties{
				OnvifProtocol: map[string]string{
					EndpointRefAddress: "456",
				},
			},
		},
		{
			Name: "testDevice3", Protocols: map[string]models.ProtocolProperties{
				OnvifProtocol: map[string]string{
					EndpointRefAddress: "789",
				},
			},
		},
	}
}

func TestOnvifDiscovery_makeDeviceMap(t *testing.T) {
	tests := []struct {
		name      string
		devices   []contract.Device
		deviceMap map[string]contract.Device
		nameCalls int
	}{
		{
			name:    "3 devices",
			devices: createTestDeviceList(),
			deviceMap: map[string]contract.Device{
				"123": {
					Name: "testDevice1", Protocols: map[string]models.ProtocolProperties{
						OnvifProtocol: map[string]string{
							EndpointRefAddress: "123",
						},
					},
				},
				"456": {
					Name: "testDevice2", Protocols: map[string]models.ProtocolProperties{
						OnvifProtocol: map[string]string{
							EndpointRefAddress: "456",
						},
					},
				},
				"789": {
					Name: "testDevice3", Protocols: map[string]models.ProtocolProperties{
						OnvifProtocol: map[string]string{
							EndpointRefAddress: "789",
						},
					},
				},
			},
			nameCalls: 4,
		},
		{
			name: "NoProtocol",
			devices: []contract.Device{
				{
					Name: "testDevice1",
					Protocols: map[string]models.ProtocolProperties{
						OnvifProtocol: map[string]string{
							EndpointRefAddress: "123",
						},
					},
				},
				{
					Name:      "testDevice2",
					Protocols: map[string]models.ProtocolProperties{},
				},
			},
			deviceMap: map[string]contract.Device{
				"123": {
					Name: "testDevice1", Protocols: map[string]models.ProtocolProperties{
						OnvifProtocol: map[string]string{
							EndpointRefAddress: "123",
						},
					},
				},
			},
			nameCalls: 2,
		},
		{
			name: "NoEndpointReference",
			devices: []contract.Device{
				{
					Name: "testDevice1",
					Protocols: map[string]models.ProtocolProperties{
						OnvifProtocol: map[string]string{
							EndpointRefAddress: "123",
						},
					},
				},
				{
					Name: "testDevice2",
					Protocols: map[string]models.ProtocolProperties{
						OnvifProtocol: map[string]string{
							EndpointRefAddress: "",
						},
					},
				},
			},
			deviceMap: map[string]contract.Device{
				"123": {
					Name: "testDevice1", Protocols: map[string]models.ProtocolProperties{
						OnvifProtocol: map[string]string{
							EndpointRefAddress: "123",
						},
					},
				},
			},
			nameCalls: 2,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			mockService, driver := createDriverWithMockService()
			mockService.On("Devices").
				Return(test.devices).Once()
			mockService.On("Name").
				Return("device-onvif-camera").Times(test.nameCalls)
			devices := driver.makeDeviceMap()
			mockService.AssertExpectations(t)

			assert.Equal(t, devices, test.deviceMap)
		})
	}
}

func TestOnvifDiscovery_discoveryFilter(t *testing.T) {
	tests := []struct {
		name              string
		devices           []contract.Device
		discoveredDevices []sdkModel.DiscoveredDevice
		filtered          []sdkModel.DiscoveredDevice
		nameCalls         int
	}{
		{
			name:              "No new devices",
			devices:           createTestDeviceList(),
			discoveredDevices: createDiscoveredList(),
			filtered:          []sdkModel.DiscoveredDevice(nil),
			nameCalls:         4,
		},
		{
			name: "All new devices",
			devices: []contract.Device{
				{
					Name: "device-onvif-camera", Protocols: map[string]models.ProtocolProperties{
						OnvifProtocol: map[string]string{
							EndpointRefAddress: "abc",
						},
					},
				},
			},
			discoveredDevices: createDiscoveredList(),
			filtered:          createDiscoveredList(),
			nameCalls:         1,
		},
		{
			name:    "new and old devices",
			devices: createTestDeviceList(),
			discoveredDevices: []sdkModel.DiscoveredDevice{
				{
					Name: "testDevice1", Protocols: map[string]models.ProtocolProperties{
						OnvifProtocol: map[string]string{
							EndpointRefAddress: "123",
						},
					},
				},
				{
					Name: "testDevice2", Protocols: map[string]models.ProtocolProperties{
						OnvifProtocol: map[string]string{
							EndpointRefAddress: "456",
						},
					},
				},
				{
					Name: "testDevice3", Protocols: map[string]models.ProtocolProperties{
						OnvifProtocol: map[string]string{
							EndpointRefAddress: "789",
						},
					},
				},
				{
					Name: "testDevice4", Protocols: map[string]models.ProtocolProperties{
						OnvifProtocol: map[string]string{
							EndpointRefAddress: "xyz",
						},
					},
				},
				{
					Name: "testDevice5", Protocols: map[string]models.ProtocolProperties{
						OnvifProtocol: map[string]string{
							EndpointRefAddress: "def",
						},
					},
				},
			},
			filtered: []sdkModel.DiscoveredDevice{
				{
					Name: "testDevice4", Protocols: map[string]models.ProtocolProperties{
						OnvifProtocol: map[string]string{
							EndpointRefAddress: "xyz",
						},
					},
				},
				{
					Name: "testDevice5", Protocols: map[string]models.ProtocolProperties{
						OnvifProtocol: map[string]string{
							EndpointRefAddress: "def",
						},
					},
				},
			},
			nameCalls: 4,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			mockService, driver := createDriverWithMockService()
			mockService.On("Devices").
				Return(test.devices).Once()
			mockService.On("Name").
				Return("device-onvif-camera").Times(test.nameCalls)
			filtered := driver.discoverFilter(test.discoveredDevices)
			mockService.AssertExpectations(t)

			assert.Equal(t, test.filtered, filtered)
		})
	}
}

func TestOnvifDiscovery_discoverFilter_mixedNewAndOldDevices(t *testing.T) {
	mockService, driver := createDriverWithMockService()
	mockService.On("Devices").
		Return(createTestDeviceList()).Once()
	mockService.On("Name").
		Return("device-onvif-camera").Times(4)
	actual := driver.discoverFilter(
		[]sdkModel.DiscoveredDevice{
			{
				Name: "testDevice1", Protocols: map[string]models.ProtocolProperties{
					OnvifProtocol: map[string]string{
						EndpointRefAddress: "123",
					},
				},
			},
			{
				Name: "testDevice2", Protocols: map[string]models.ProtocolProperties{
					OnvifProtocol: map[string]string{
						EndpointRefAddress: "456",
					},
				},
			},
			{
				Name: "testDevice3", Protocols: map[string]models.ProtocolProperties{
					OnvifProtocol: map[string]string{
						EndpointRefAddress: "789",
					},
				},
			},
			{
				Name: "testDevice4", Protocols: map[string]models.ProtocolProperties{
					OnvifProtocol: map[string]string{
						EndpointRefAddress: "xyz",
					},
				},
			},
			{
				Name: "testDevice5", Protocols: map[string]models.ProtocolProperties{
					OnvifProtocol: map[string]string{
						EndpointRefAddress: "def",
					},
				},
			},
		},
	)
	mockService.AssertExpectations(t)

	assert.Equal(t, actual, []sdkModel.DiscoveredDevice{
		{
			Name: "testDevice4", Protocols: map[string]models.ProtocolProperties{
				OnvifProtocol: map[string]string{
					EndpointRefAddress: "xyz",
				},
			},
		},
		{
			Name: "testDevice5", Protocols: map[string]models.ProtocolProperties{
				OnvifProtocol: map[string]string{
					EndpointRefAddress: "def",
				},
			},
		},
	})
}

// func TestOnvifDiscovery_updateExistingDevice(t *testing.T) {
// 	mockService, driver := createDriverWithMockService()
// 	mockService.On("UpdateDevice", models.Device{
// 		Protocols: map[string]contract.ProtocolProperties{
// 			OnvifProtocol: map[string]string{
// 				"Address":  "5.6.7.8",
// 				"Port":     "2",
// 				"LastSeen": time.Now().Format(time.UnixDate),
// 			},
// 		},
// 	}).Return(nil).Once()
// 	err := driver.updateExistingDevice(
// 		contract.Device{
// 			Protocols: map[string]contract.ProtocolProperties{
// 				OnvifProtocol: map[string]string{
// 					"Address": "1.2.3.4",
// 					"Port":    "1",
// 				},
// 			},
// 		}, sdkModel.DiscoveredDevice{
// 			Protocols: map[string]contract.ProtocolProperties{
// 				OnvifProtocol: map[string]string{
// 					"Address": "5.6.7.8",
// 					"Port":    "2",
// 				},
// 			},
// 		})
// 	driver.lc.Info("Helo")
// 	mockService.AssertExpectations(t)
// 	require.NoError(t, err)
// }

// func TestDriver_createDiscoveredDevice(t *testing.T) {
// 	type fields struct {
// 		lc               logger.LoggingClient
// 		asynchCh         chan<- *sdkModel.AsyncValues
// 		deviceCh         chan<- []sdkModel.DiscoveredDevice
// 		sdkService       SDKService
// 		onvifClients     map[string]*OnvifClient
// 		clientsMu        *sync.RWMutex
// 		config           *ServiceConfig
// 		configMu         *sync.RWMutex
// 		addedWatchers    bool
// 		watchersMu       sync.Mutex
// 		macAddressMapper *MACAddressMapper
// 		debounceTimer    *time.Timer
// 		debounceMu       sync.Mutex
// 		taskCh           chan struct{}
// 		wg               sync.WaitGroup
// 	}
// 	type args struct {
// 		onvifDevice onvif.Device
// 	}
// 	tests := []struct {
// 		name    string
// 		fields  fields
// 		args    args
// 		want    sdkModel.DiscoveredDevice
// 		wantErr bool
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			d := &Driver{
// 				lc:               tt.fields.lc,
// 				asynchCh:         tt.fields.asynchCh,
// 				deviceCh:         tt.fields.deviceCh,
// 				sdkService:       tt.fields.sdkService,
// 				onvifClients:     tt.fields.onvifClients,
// 				clientsMu:        tt.fields.clientsMu,
// 				config:           tt.fields.config,
// 				configMu:         tt.fields.configMu,
// 				addedWatchers:    tt.fields.addedWatchers,
// 				watchersMu:       tt.fields.watchersMu,
// 				macAddressMapper: tt.fields.macAddressMapper,
// 				debounceTimer:    tt.fields.debounceTimer,
// 				debounceMu:       tt.fields.debounceMu,
// 				taskCh:           tt.fields.taskCh,
// 				wg:               tt.fields.wg,
// 			}
// 			got, err := d.createDiscoveredDevice(tt.args.onvifDevice)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("Driver.createDiscoveredDevice() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("Driver.createDiscoveredDevice() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }
