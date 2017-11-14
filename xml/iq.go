/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

import "unsafe"

type IQ struct {
	Element
}

func NewIQFromElement(element Element) (*IQ, error) {
	iq := &IQ{}
	iq.p = unsafe.Pointer(element.shared())
	iq.shadowed = 0
	return iq, nil
}
