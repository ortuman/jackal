package c2smodel

import (
	"strconv"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
)

// Resource represents a resource entity.
type Resource struct {
	InstanceID string
	JID        *jid.JID
	Presence   *stravaganza.Presence
	Info       Info
}

// IsAvailable returns presence available value.
func (r *Resource) IsAvailable() bool {
	if r.Presence != nil {
		return r.Presence.IsAvailable()
	}
	return false
}

// Priority returns resource presence priority.
func (r *Resource) Priority() int8 {
	if r.Presence != nil {
		return r.Presence.Priority()
	}
	return 0
}

// State represents a client-to-server stream state.
type State struct {
	State    uint32
	JID      *jid.JID
	Presence *stravaganza.Presence
	Info     Info
}

// Info represents a C2S resource user info.
type Info struct {
	m map[string]string
}

// InfoFromMap creates a new C2S info based on m map.
func InfoFromMap(m map[string]string) Info {
	newM := make(map[string]string, len(m))
	for k, v := range m {
		newM[k] = v
	}
	return Info{
		m: newM,
	}
}

// String returns string value associated to k key.
func (i Info) String(k string) string {
	return i.m[k]
}

// Bool returns boolean value associated to k key.
func (i Info) Bool(k string) bool {
	ok, _ := strconv.ParseBool(i.m[k])
	return ok
}

// Int returns integer value associated to k key.
func (i Info) Int(k string) int64 {
	in, _ := strconv.ParseInt(i.m[k], 10, 64)
	return in
}

// Float returns integer value associated to k key.
func (i Info) Float(k string) float64 {
	f, _ := strconv.ParseFloat(i.m[k], 64)
	return f
}

// Map returns C2S info map representation.
func (i Info) Map() map[string]string {
	outM := make(map[string]string, len(i.m))
	for k, v := range i.m {
		outM[k] = v
	}
	return outM
}
