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

// copied from https://github.com/etcd-io/etcd/blob/master/etcdctl/ctlv3/command/error.go

package command

import (
	"fmt"
	"os"
)

const (
	// ExitSuccess is returned in case command was correctly executed.
	ExitSuccess = iota

	// ExitError is catchall for general errors.
	ExitError

	// ExitBadArgs is returned in case passed arguments are invalid.
	ExitBadArgs = 128
)

// ExitWithError exits application printing error value to stderr.
func ExitWithError(code int, err error) {
	_, _ = fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(code)
}
