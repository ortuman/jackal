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

package router

import "errors"

var (
	// ErrNotExistingAccount will be returned by Route method if destination user does not exist.
	ErrNotExistingAccount = errors.New("router: account does not exist")

	// ErrResourceNotFound will be returned by Route method if destination resource does not match any of user's available resources.
	ErrResourceNotFound = errors.New("router: resource not found")

	// ErrUserNotAvailable will be returned by Route method in case no available resource with non negative priority was found.
	ErrUserNotAvailable = errors.New("router: user not available")

	// ErrRemoteServerNotFound will be returned by Route method if couldn't establish a connection to the remote server.
	ErrRemoteServerNotFound = errors.New("router: remote server not found")

	// ErrRemoteServerTimeout will be returned by Route method if maximum amount of time to establish remote connection
	// was reached.
	ErrRemoteServerTimeout = errors.New("router: remote server timeout")
)
