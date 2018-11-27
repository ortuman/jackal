/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package model

import (
	"encoding/gob"
)

// GobSerializer represents a Gob serializable entity.
type GobSerializer interface {
	ToGob(enc *gob.Encoder)
}

// GobDeserializer represents a Gob deserializable entity.
type GobDeserializer interface {
	FromGob(dec *gob.Decoder) error
}
