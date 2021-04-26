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

package instance

import (
	"os"
	"sync"

	"github.com/google/uuid"
)

const (
	envInstanceID = "JACKAL_INSTANCE_ID"
)

var (
	once   sync.Once
	instID string
)

// ID returns local instance identifier.
func ID() string {
	loadInstance()
	return instID
}

func loadInstance() {
	once.Do(func() {
		id := os.Getenv(envInstanceID)
		if len(id) == 0 {
			id = uuid.New().String() // if unspecified, assign UUID identifier
		}
		instID = id
	})
}
