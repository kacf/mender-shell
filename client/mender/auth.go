// Copyright 2020 Northern.tech AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package mender

import (
	"errors"
	"time"

	"github.com/mendersoftware/mender-shell/client/dbus"
)

// DbBus constants for the Mender Authentication Manager
const (
	DBusObjectName                       = "io.mender.AuthenticationManager"
	DBusObjectPath                       = "/io/mender/AuthenticationManager"
	DBusInterfaceName                    = "io.mender.Authentication1"
	DBusMethodNameGetJwtToken            = "GetJwtToken"
	DBusMethodNameFetchJwtToken          = "FetchJwtToken"
	DBusSignalNameValidJwtTokenAvailable = "ValidJwtTokenAvailable"
	DBusMethodTimeoutInSeconds           = 5
)

var timeout = 10 * time.Second
var errFetchTokenFailed = errors.New("FetchJwtToken failed")

// AuthClient is the interface for the Mender Authentication Manager clilents
type AuthClient interface {
	// Connect to the Mender client interface
	Connect(objectName, objectPath, interfaceName string) error
	// GetJWTToken returns a device JWT token
	GetJWTToken() (string, error)
	// FetchJWTToken schedules the fetching of a new device JWT token
	FetchJWTToken() (bool, error)
	// WaitForValidJWTTokenAvailable synchronously waits for the ValidJwtTokenAvailable signal
	WaitForValidJWTTokenAvailable() error
	// FetchAndGetJWTToken fetches a new JWT token and returns it
	FetchAndGetJWTToken() (string, error)
}

// AuthClientDBUS is the implementation of the client for the Mender
// Authentication Manager which communicates using DBUS
type AuthClientDBUS struct {
	dbusAPI          dbus.DBusAPI
	dbusConnection   dbus.Handle
	authManagerProxy dbus.Handle
}

// NewAuthClient returns a new AuthClient
func NewAuthClient(dbusAPI dbus.DBusAPI) (AuthClient, error) {
	if dbusAPI == nil {
		var err error
		dbusAPI, err = dbus.GetDBusAPI()
		if err != nil {
			return nil, err
		}
	}
	return &AuthClientDBUS{
		dbusAPI: dbusAPI,
	}, nil
}

// Connect to the Mender client interface
func (a *AuthClientDBUS) Connect(objectName, objectPath, interfaceName string) error {
	dbusConnection, err := a.dbusAPI.BusGet(dbus.GBusTypeSystem)
	if err != nil {
		return err
	}
	authManagerProxy, err := a.dbusAPI.BusProxyNew(dbusConnection, objectName, objectPath, interfaceName)
	if err != nil {
		return err
	}
	a.dbusConnection = dbusConnection
	a.authManagerProxy = authManagerProxy
	return nil
}

// GetJWTToken returns a device JWT token
func (a *AuthClientDBUS) GetJWTToken() (string, error) {
	response, err := a.dbusAPI.BusProxyCall(a.authManagerProxy, DBusMethodNameGetJwtToken, nil, DBusMethodTimeoutInSeconds)
	if err != nil {
		return "", err
	}
	return response.GetString(), nil
}

// FetchJWTToken schedules the fetching of a new device JWT token
func (a *AuthClientDBUS) FetchJWTToken() (bool, error) {
	response, err := a.dbusAPI.BusProxyCall(a.authManagerProxy, DBusMethodNameFetchJwtToken, nil, DBusMethodTimeoutInSeconds)
	if err != nil {
		return false, err
	}
	return response.GetBoolean(), nil
}

// WaitForValidJWTTokenAvailable synchronously waits for the ValidJwtTokenAvailable signal
func (a *AuthClientDBUS) WaitForValidJWTTokenAvailable() error {
	return a.dbusAPI.WaitForSignal(DBusSignalNameValidJwtTokenAvailable, timeout)
}

// FetchAndGetJWTToken fetches a new JWT token and returns it
func (a *AuthClientDBUS) FetchAndGetJWTToken() (string, error) {
	fetch, err := a.FetchJWTToken()
	if err != nil {
		return "", err
	} else if fetch == false {
		return "", errFetchTokenFailed
	}
	err = a.WaitForValidJWTTokenAvailable()
	if err != nil {
		return "", err
	}
	return a.GetJWTToken()
}
