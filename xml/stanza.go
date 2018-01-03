/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

type Stanza interface {
	Serializable
	ToJID() *JID
	FromJID() *JID
}
