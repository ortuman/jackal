/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

type Stanza interface {
	ToJID() *JID
	FromJID() *JID
}
