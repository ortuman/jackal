/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

type IQ struct {
	Element
}

func NewIQ(e *Element) (*IQ, error) {
	iq := &IQ{}
	return iq, nil
}
