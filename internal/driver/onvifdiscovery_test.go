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

// func TestOnvifDiscovery_discoveryFilter(t *testing.T) {
// 	tests := []struct {
// 		{
// 			name:
// 			discoveredDevices []
// 		}
// 	}{

// 	}
// }
func TestOnvifDiscovery_discoverFilter_noNewDevices(t *testing.T) {
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
		},
	)
	mockService.AssertExpectations(t)

	assert.Equal(t, actual, []sdkModel.DiscoveredDevice(nil))
}

func TestOnvifDiscovery_discoverFilter_newDevices(t *testing.T) {
	mockService, driver := createDriverWithMockService()
	mockService.On("Devices").
		Return(nil).Once()
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
		},
	)
	mockService.AssertExpectations(t)

	assert.Equal(t, actual, []sdkModel.DiscoveredDevice{
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
	})
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
