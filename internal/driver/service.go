//
// Copyright (C) 2022 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/service"
	"github.com/edgexfoundry/go-mod-bootstrap/v2/bootstrap/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/models"
	"net/http"
	"sync"
)

// SDKService wraps an EdgeX SDK service.DeviceService so that it can be easily mocked in tests.
type SDKService interface {
	// ------------------------------------------
	// service.DeviceService inherited functions
	// ------------------------------------------

	// AddDeviceAutoEvent adds a new AutoEvent to the Device with given name
	AddDeviceAutoEvent(deviceName string, event models.AutoEvent) error

	// RemoveDeviceAutoEvent removes an AutoEvent from the Device with given name
	RemoveDeviceAutoEvent(deviceName string, event models.AutoEvent) error

	// AddDevice adds a new Device to the Device Service and Core Metadata
	// Returns new Device id or non-nil error.
	AddDevice(device models.Device) (string, error)

	// Devices return all managed Devices from cache
	Devices() []models.Device

	// GetDeviceByName returns the Device by its name if it exists in the cache, or returns an error.
	GetDeviceByName(name string) (models.Device, error)

	// RemoveDeviceByName removes the specified Device by name from the cache and ensures that the
	// instance in Core Metadata is also removed.
	RemoveDeviceByName(name string) error

	// UpdateDevice updates the Device in the cache and ensures that the
	// copy in Core Metadata is also updated.
	UpdateDevice(device models.Device) error

	// UpdateDeviceOperatingState updates the Device's OperatingState with given name
	// in Core Metadata and device service cache.
	UpdateDeviceOperatingState(deviceName string, state string) error

	// AddDeviceProfile adds a new DeviceProfile to the Device Service and Core Metadata
	// Returns new DeviceProfile id or non-nil error.
	AddDeviceProfile(profile models.DeviceProfile) (string, error)

	// DeviceProfiles return all managed DeviceProfiles from cache
	DeviceProfiles() []models.DeviceProfile

	// GetProfileByName returns the Profile by its name if it exists in the cache, or returns an error.
	GetProfileByName(name string) (models.DeviceProfile, error)

	// RemoveDeviceProfileByName removes the specified DeviceProfile by name from the cache and ensures that the
	// instance in Core Metadata is also removed.
	RemoveDeviceProfileByName(name string) error

	// UpdateDeviceProfile updates the DeviceProfile in the cache and ensures that the
	// copy in Core Metadata is also updated.
	UpdateDeviceProfile(profile models.DeviceProfile) error

	// DeviceCommand retrieves the specific DeviceCommand instance from cache according to
	// the Device name and Command name
	DeviceCommand(deviceName string, commandName string) (models.DeviceCommand, bool)

	// DeviceResource retrieves the specific DeviceResource instance from cache according to
	// the Device name and Device Resource name
	DeviceResource(deviceName string, deviceResource string) (models.DeviceResource, bool)

	// AddProvisionWatcher adds a new Watcher to the cache and Core Metadata
	// Returns new Watcher id or non-nil error.
	AddProvisionWatcher(watcher models.ProvisionWatcher) (string, error)

	// ProvisionWatchers return all managed Watchers from cache
	ProvisionWatchers() []models.ProvisionWatcher

	// GetProvisionWatcherByName returns the Watcher by its name if it exists in the cache, or returns an error.
	GetProvisionWatcherByName(name string) (models.ProvisionWatcher, error)

	// RemoveProvisionWatcher removes the specified Watcher by name from the cache and ensures that the
	// instance in Core Metadata is also removed.
	RemoveProvisionWatcher(name string) error

	// UpdateProvisionWatcher updates the Watcher in the cache and ensures that the
	// copy in Core Metadata is also updated.
	UpdateProvisionWatcher(watcher models.ProvisionWatcher) error

	// Name returns the name of this Device Service
	Name() string

	// Version returns the version number of this Device Service
	Version() string

	// AsyncReadings returns a bool value to indicate whether the asynchronous reading is enabled.
	AsyncReadings() bool

	// DeviceDiscovery returns a bool value to indicate whether device discovery is enabled.
	DeviceDiscovery() bool

	// AddRoute allows leveraging the existing internal web server to add routes specific to Device Service.
	AddRoute(route string, handler func(http.ResponseWriter, *http.Request), methods ...string) error

	// Stop shuts down the Service
	Stop(force bool)

	// LoadCustomConfig uses the Config Processor from go-mod-bootstrap to attempt to load service's
	// custom configuration. It uses the same command line flags to process the custom config in the same manner
	// as the standard configuration.
	LoadCustomConfig(customConfig service.UpdatableConfig, sectionName string) error

	// ListenForCustomConfigChanges uses the Config Processor from go-mod-bootstrap to attempt to listen for
	// changes to the specified custom configuration section. LoadCustomConfig must be called previously so that
	// the instance of sdkService.configProcessor has already been set.
	ListenForCustomConfigChanges(configToWatch interface{}, sectionName string, changedCallback func(interface{})) error

	// ------------------------------------------
	// Custom managed functionality or macros
	// ------------------------------------------

	// GetSecretProvider returns the interfaces.SecretProvider. The name chosen to avoid conflicts
	// with service.DeviceService.SecretProvider struct field.
	GetSecretProvider() interfaces.SecretProvider

	// UpdateDeviceWithLock applies a thread-safe update of a device. The input is the device name and a lambda
	// function to call with the most up-to-date device object. The lambda shall return a bool whether the device
	// has been modified.
	UpdateDeviceWithLock(deviceName string, updateFunc func(device *models.Device) bool) error
}

// DeviceSDKService is the native implementation of ServiceWrapper that uses the actual
// service.DeviceService struct.
type DeviceSDKService struct {
	*service.DeviceService

	devicesMu sync.RWMutex
}

// GetSecretProvider returns the SecretProvider
func (s *DeviceSDKService) GetSecretProvider() interfaces.SecretProvider {
	return s.SecretProvider
}

func (s *DeviceSDKService) UpdateDeviceWithLock(deviceName string, updateFunc func(device *models.Device) bool) error {
	s.devicesMu.Lock()
	defer s.devicesMu.Unlock()

	// we already have the lock, so call it directly
	device, err := s.DeviceService.GetDeviceByName(deviceName)
	if err != nil {
		return err
	}

	changed := updateFunc(&device)
	if changed {
		return s.DeviceService.UpdateDevice(device)
	}
	return nil
}

func (s *DeviceSDKService) GetDeviceByName(name string) (models.Device, error) {
	s.devicesMu.RLock()
	defer s.devicesMu.RUnlock()

	return s.DeviceService.GetDeviceByName(name)
}
