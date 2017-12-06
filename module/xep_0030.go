/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package module

type discoItem struct {
	jid  string
	name string
	node string
}

type discoIdentity struct {
	category string
	tp       string
	name     string
}
