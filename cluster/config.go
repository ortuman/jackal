/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

// Config represents an cluster configuration.
type Config struct {
	Name     string   `json:"name"`
	BindPort int      `json:"bind_port"`
	Hosts    []string `json:"hosts"`
}
