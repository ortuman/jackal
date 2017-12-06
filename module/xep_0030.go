/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package module

type DiscoItem struct {
	Jid  string
	Name string
	Node string
}

type DiscoIdentity struct {
	Category string
	Type     string
	Name     string
}
