/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import (
	"time"
)

const (
	delayNamespace = "urn:xmpp:delay"
)

// Delay attaches element's Delayed Delivery information.
func (m *MutableElement) Delay(from string, text string) {
	d := NewElementNamespace("delay", delayNamespace)
	if len(from) > 0 {
		d.SetAttribute("from", from)
	}
	t := time.Now()
	d.SetAttribute("stamp", t.Format("2006-01-02T15:04:05Z"))

	if len(text) > 0 {
		d.SetText(text)
	}
	m.AppendElement(d)
}
