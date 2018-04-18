/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import (
	"encoding/gob"
	"fmt"
	"github.com/ortuman/jackal/pool"
	"io"
)

var bufPool = pool.NewBufferPool()

// ErrorType represents an 'error' stanza type.
const ErrorType = "error"

// XElement represents an XML node element.
type XElement interface {
	fmt.Stringer

	Name() string
	Attributes() AttributeSet
	Elements() ElementSet

	Text() string

	ID() string
	Namespace() string
	Language() string
	Version() string
	From() string
	To() string
	Type() string

	Error() XElement

	ToXML(w io.Writer, includeClosing bool)
	ToGob(enc *gob.Encoder)
}
