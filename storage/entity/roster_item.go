/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package entity

type RosterItem struct {
	Jid          string
	Name         string
	Subscription string
	Ask          bool
	Groups       []string
}
