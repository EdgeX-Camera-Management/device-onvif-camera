// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2022 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"encoding/base64"
	"fmt"
	"sync"
	"testing"

	"github.com/IOTechSystems/onvif/device"
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/interfaces/mocks"
	sdkModel "github.com/edgexfoundry/device-sdk-go/v2/pkg/models"

	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/models"
	contract "github.com/edgexfoundry/go-mod-core-contracts/v2/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDeviceName = "test-device"
)

func createDriverWithMockService() (*Driver, *mocks.DeviceServiceSDK) {

	mockService := &mocks.DeviceServiceSDK{}
	driver := &Driver{sdkService: mockService, lc: logger.MockLogger{}}
	return driver, mockService
}

func createTestDevice() models.Device {
	return models.Device{Name: testDeviceName, Protocols: map[string]models.ProtocolProperties{
		OnvifProtocol: map[string]string{
			DeviceStatus: Unreachable,
		},
	}}
}

func createTestDeviceWithProtocols(protocols map[string]models.ProtocolProperties) models.Device {
	return models.Device{Name: testDeviceName, Protocols: protocols}
}

func TestParametersFromURLRawQuery(t *testing.T) {
	parameters := `{ "ProfileToken": "Profile_1" }`
	base64EncodedStr := base64.StdEncoding.EncodeToString([]byte(parameters))
	req := sdkModel.CommandRequest{
		Attributes: map[string]interface{}{
			URLRawQuery: fmt.Sprintf("%s=%s", jsonObject, base64EncodedStr),
		},
	}
	data, err := parametersFromURLRawQuery(req)
	require.NoError(t, err)
	assert.Equal(t, parameters, string(data))
}

// TestAddressAndPort verifies splitting of address and port from a given string.
func TestAddressAndPort(t *testing.T) {

	tests := []struct {
		input           string
		expectedAddress string
		expectedPort    string
	}{
		{
			input:           "localhost:80",
			expectedAddress: "localhost",
			expectedPort:    "80",
		},
		{
			input:           "localhost",
			expectedAddress: "localhost",
			expectedPort:    "80",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.input, func(t *testing.T) {
			resultAddress, resultPort := addressAndPort(test.input)
			assert.Equal(t, test.expectedAddress, resultAddress)
			assert.Equal(t, test.expectedPort, resultPort)
		})
	}
}

// TestUpdateDevice: verifies proper updating of device information
func TestUpdateDevice(t *testing.T) {
	driver, mockService := createDriverWithMockService()
	tests := []struct {
		device  models.Device
		devInfo *device.GetDeviceInformationResponse

		expectedDevice models.Device
		errorExpected  bool

		removeDeviceFailExpected bool
	}{
		{
			device: contract.Device{
				Name: "testName",
			},
			devInfo: &device.GetDeviceInformationResponse{
				Manufacturer:    "Intel",
				Model:           "SimCamera",
				FirmwareVersion: "2.5a",
				SerialNumber:    "9a32410c",
				HardwareId:      "1.0",
			},
		},
		{
			device: contract.Device{
				Name: "unknown_unknown_device",
			},
			devInfo: &device.GetDeviceInformationResponse{
				Manufacturer:    "Intel",
				Model:           "SimCamera",
				FirmwareVersion: "2.5a",
				SerialNumber:    "9a32410c",
				HardwareId:      "1.0",
			},
			expectedDevice: contract.Device{
				Name: "Intel-SimCamera-",
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.device.Name, func(t *testing.T) {

			if test.removeDeviceFailExpected {
				mockService.On("RemoveDeviceByName", test.device.Name).Return("error").Once()
			} else {
				mockService.On("RemoveDeviceByName", test.device.Name).Return(nil).Once()
			}
			mockService.On("RemoveDeviceByName", test.device.Name).Return(nil).Once()
			mockService.On("AddDevice", test.expectedDevice).Return(test.expectedDevice.Name, nil).Once()
			mockService.On("UpdateDevice", test.device).Return(nil).Once()

			err := driver.updateDevice(test.device, test.devInfo)

			if test.errorExpected {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestDriver_RemoveDevice(t *testing.T) {
	driver, mockService := createDriverWithMockService()
	driver.asynchCh = make(chan *sdkModel.AsyncValues, 1)
	driver.clientsMu = new(sync.RWMutex)
	driver.configMu = new(sync.RWMutex)
	driver.onvifClients = make(map[string]*OnvifClient)

	tests := []struct {
		name       string
		deviceName string
		protocols  map[string]models.ProtocolProperties
		wantErr    bool
	}{
		{
			name:       "control plane device",
			deviceName: "device-onvif-camera",
			protocols:  map[string]models.ProtocolProperties{},
		},
		{
			name:       "simple device",
			deviceName: "my-device",
			protocols:  map[string]models.ProtocolProperties{},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			mockService.On("Name").Return(test.deviceName)

			err := driver.RemoveDevice(test.deviceName, test.protocols)
			if test.wantErr {
				require.Error(t, err)
			}
			mockService.AssertExpectations(t)
		})
	}
}
