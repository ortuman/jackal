// Copyright 2020 The jackal Authors
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

package pepper

import (
	"errors"
	"fmt"
)

var minKeyLength = 24

type Config struct {
	Keys  map[string]string `fig:"keys"`
	UseID string            `fig:"use"`
}

// Keys contains all configured pepper keys.
type Keys struct {
	ks    map[string]string
	useID string
}

// NewKeys returns an initialized set of pepper keys.
func NewKeys(cfg Config) (*Keys, error) {
	if len(cfg.Keys) == 0 {
		return nil, errors.New("pepper: no pepper keys defined")
	}
	for keyID, k := range cfg.Keys {
		if len(k) < minKeyLength {
			return nil, fmt.Errorf("pepper: key %s must be at least %d characters", keyID, minKeyLength)
		}
	}
	_, ok := cfg.Keys[cfg.UseID]
	if !ok {
		return nil, fmt.Errorf("pepper: active key not found: %s", cfg.UseID)
	}
	return &Keys{ks: cfg.Keys, useID: cfg.UseID}, nil
}

// GetKey returns pepper associated to an identifier.
func (k *Keys) GetKey(pepperID string) string {
	return k.ks[pepperID]
}

// GetActiveKey returns active pepper value.
func (k *Keys) GetActiveKey() string {
	return k.ks[k.useID]
}

// GetActiveID returns active pepper identifier.
func (k *Keys) GetActiveID() string {
	return k.useID
}
