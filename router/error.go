/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import "github.com/pkg/errors"

var (
	// ErrNotExistingAccount will be returned by Route method
	// if destination user does not exist.
	ErrNotExistingAccount = errors.New("router: account does not exist")

	// ErrResourceNotFound will be returned by Route method
	// if destination resource does not match any of user's available resources.
	ErrResourceNotFound = errors.New("router: resource not found")

	// ErrNotAuthenticated will be returned by Route method if
	// destination user is not available at this moment.
	ErrNotAuthenticated = errors.New("router: user not authenticated")

	// ErrBlockedJID will be returned by Route method if
	// destination jid matches any of the user's blocked jid.
	ErrBlockedJID = errors.New("router: destination jid is blocked")

	// ErrFailedRemoteConnect will be returned by Route method if
	// couldn't establish a connection to the remote server.
	ErrFailedRemoteConnect = errors.New("router: failed remote connection")
)
