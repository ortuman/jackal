// Copyright 2022 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package instance

import (
	"errors"
	"net"
	"os"

	"github.com/google/uuid"
)

const (
	envInstanceID = "JACKAL_INSTANCE_ID"
	envHostName   = "JACKAL_HOSTNAME"
)

var (
	instID, hostIP string
)

var (
	readCachedResults  = true
	interfaceAddresses = net.InterfaceAddrs
)

func init() {
	instID = getID()
	hostIP = getHostname()
}

// ID returns local instance identifier.
func ID() string {
	if readCachedResults {
		return instID
	}
	return getID()
}

// Hostname returns local instance host name.
func Hostname() string {
	if readCachedResults {
		return hostIP
	}
	return getHostname()
}

func getID() string {
	id := os.Getenv(envInstanceID)
	if len(id) == 0 {
		return uuid.New().String() // if unspecified, assign UUID identifier
	}
	return id
}

func getHostname() string {
	fqdn := os.Getenv(envHostName)
	if len(fqdn) > 0 {
		return fqdn
	}
	hn, err := getLocalHostname()
	if err == nil && len(hn) > 0 {
		return hn
	}
	return "localhost" // fallback to 'localhost' ip
}

func getLocalHostname() (string, error) {
	addresses, err := interfaceAddresses()
	if err != nil {
		return "", err
	}

	for _, addr := range addresses {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}
	return "", errors.New("instance: failed to get local ip")
}
