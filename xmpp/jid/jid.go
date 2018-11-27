/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package jid

import (
	"bytes"
	"encoding/gob"
	"errors"
	"net"
	"strings"
	"unicode/utf8"

	"github.com/ortuman/jackal/pool"
	"golang.org/x/net/idna"
	"golang.org/x/text/secure/precis"
)

var bufPool = pool.NewBufferPool()

// MatchingOptions represents a matching jid mask.
type MatchingOptions int8

const (
	// MatchesNode indicates that left and right operand has same node value.
	MatchesNode = MatchingOptions(1)

	// MatchesDomain indicates that left and right operand has same domain value.
	MatchesDomain = MatchingOptions(2)

	// MatchesResource indicates that left and right operand has same resource value.
	MatchesResource = MatchingOptions(4)

	// MatchesBare indicates that left and right operand has same node and domain value.
	MatchesBare = MatchesNode | MatchesDomain
)

// JID represents an XMPP address (JID).
// A JID is made up of a node (generally a username), a domain, and a resource.
// The node and resource are optional; domain is required.
type JID struct {
	node     string
	domain   string
	resource string
}

// New constructs a JID given a user, domain, and resource.
// This construction allows the caller to specify if stringprep should be applied or not.
func New(node, domain, resource string, skipStringPrep bool) (*JID, error) {
	if skipStringPrep {
		return &JID{
			node:     node,
			domain:   domain,
			resource: resource,
		}, nil
	}
	return stringPrep(node, domain, resource)
}

// NewWithString constructs a JID from it's string representation.
// This construction allows the caller to specify if stringprep should be applied or not.
func NewWithString(str string, skipStringPrep bool) (*JID, error) {
	if len(str) == 0 {
		return &JID{}, nil
	}
	var node, domain, resource string

	atIndex := strings.Index(str, "@")
	slashIndex := strings.Index(str, "/")

	// node
	if atIndex > 0 {
		node = str[0:atIndex]
	}

	// domain
	if atIndex+1 == len(str) {
		return nil, errors.New("JID with empty domain not valid")
	}
	if atIndex < 0 {
		if slashIndex > 0 {
			domain = str[0:slashIndex]
		} else {
			domain = str
		}
	} else {
		if slashIndex > 0 {
			domain = str[atIndex+1 : slashIndex]
		} else {
			domain = str[atIndex+1:]
		}
	}

	// resource
	if slashIndex > 0 {
		if slashIndex+1 < len(str) {
			resource = str[slashIndex+1:]
		} else {
			return nil, errors.New("JID resource must not be empty")
		}
	}
	return New(node, domain, resource, skipStringPrep)
}

// Node returns the node, or empty string if this JID does not contain node information.
func (j *JID) Node() string {
	return j.node
}

// Domain returns the domain.
func (j *JID) Domain() string {
	return j.domain
}

// Resource returns the resource, or empty string if this JID does not contain resource information.
func (j *JID) Resource() string {
	return j.resource
}

// ToBareJID returns the JID equivalent of the bare JID, which is the JID with resource information removed.
func (j *JID) ToBareJID() *JID {
	if len(j.node) == 0 {
		return &JID{node: "", domain: j.domain, resource: ""}
	}
	return &JID{node: j.node, domain: j.domain, resource: ""}
}

// IsServer returns true if instance is a server JID.
func (j *JID) IsServer() bool {
	return len(j.node) == 0
}

// IsBare returns true if instance is a bare JID.
func (j *JID) IsBare() bool {
	return len(j.node) > 0 && len(j.resource) == 0
}

// IsFull returns true if instance is a full JID.
func (j *JID) IsFull() bool {
	return len(j.resource) > 0
}

// IsFullWithServer returns true if instance is a full server JID.
func (j *JID) IsFullWithServer() bool {
	return len(j.node) == 0 && len(j.resource) > 0
}

// IsFullWithUser returns true if instance is a full client JID.
func (j *JID) IsFullWithUser() bool {
	return len(j.node) > 0 && len(j.resource) > 0
}

// Matches returns true if two JID's are equivalent.
func (j *JID) Matches(j2 *JID, options MatchingOptions) bool {
	if (options&MatchesNode) > 0 && j.node != j2.node {
		return false
	}
	if (options&MatchesDomain) > 0 && j.domain != j2.domain {
		return false
	}
	if (options&MatchesResource) > 0 && j.resource != j2.resource {
		return false
	}
	return true
}

// String returns a string representation of the JID.
func (j *JID) String() string {
	buf := bufPool.Get()
	defer bufPool.Put(buf)
	if len(j.node) > 0 {
		buf.WriteString(j.node)
		buf.WriteString("@")
	}
	buf.WriteString(j.domain)
	if len(j.resource) > 0 {
		buf.WriteString("/")
		buf.WriteString(j.resource)
	}
	return buf.String()
}

// FromGob deserializes a JID entity from it's gob binary representation.
func (j *JID) FromGob(dec *gob.Decoder) error {
	dec.Decode(&j.node)
	dec.Decode(&j.domain)
	dec.Decode(&j.resource)
	return nil
}

// ToGob converts a JID entity to it's gob binary representation.
func (j *JID) ToGob(enc *gob.Encoder) {
	enc.Encode(&j.node)
	enc.Encode(&j.domain)
	enc.Encode(&j.resource)
}

func stringPrep(node, domain, resource string) (*JID, error) {
	// Ensure that parts are valid UTF-8 (and short circuit the rest of the
	// process if they're not). We'll check the domain after performing
	// the IDNA ToUnicode operation.
	if !utf8.ValidString(node) || !utf8.ValidString(resource) {
		return nil, errors.New("JID contains invalid UTF-8")
	}

	// RFC 7622 §3.2.1.  Preparation
	//
	//    An entity that prepares a string for inclusion in an XMPP domain
	//    slot MUST ensure that the string consists only of Unicode code points
	//    that are allowed in NR-LDH labels or U-labels as defined in
	//    [RFC5890].  This implies that the string MUST NOT include A-labels as
	//    defined in [RFC5890]; each A-label MUST be converted to a U-label
	//    during preparation of a string for inclusion in a domain slot.
	var err error
	domain, err = idna.ToUnicode(domain)
	if err != nil {
		return nil, err
	}
	if !utf8.ValidString(domain) {
		return nil, errors.New("domain contains invalid UTF-8")
	}

	// RFC 7622 §3.2.2.  Enforcement
	//
	//   An entity that performs enforcement in XMPP domain slots MUST
	//   prepare a string as described in Section 3.2.1 and MUST also apply
	//   the normalization, case-mapping, and width-mapping rules defined in
	//   [RFC5892].
	//
	var nodelen int
	data := make([]byte, 0, len(node)+len(domain)+len(resource))

	if node != "" {
		data, err = precis.UsernameCaseMapped.Append(data, []byte(node))
		if err != nil {
			return nil, err
		}
		nodelen = len(data)
	}
	data = append(data, []byte(domain)...)

	if resource != "" {
		data, err = precis.OpaqueString.Append(data, []byte(resource))
		if err != nil {
			return nil, err
		}
	}
	if err := commonChecks(data[:nodelen], domain, data[nodelen+len(domain):]); err != nil {
		return nil, err
	}
	return &JID{
		node:     string(data[:nodelen]),
		domain:   string(data[nodelen : nodelen+len(domain)]),
		resource: string(data[nodelen+len(domain):]),
	}, nil
}

func commonChecks(node []byte, domain string, resource []byte) error {
	l := len(node)
	if l > 1023 {
		return errors.New("node must be smaller than 1024 bytes")
	}

	// RFC 7622 §3.3.1 provides a small table of characters which are still not
	// allowed in node's even though the IdentifierClass base class and the
	// UsernameCaseMapped profile don't forbid them; disallow them here.
	if bytes.ContainsAny(node, `"&'/:<>@`) {
		return errors.New("node contains forbidden characters")
	}

	l = len(resource)
	if l > 1023 {
		return errors.New("resource must be smaller than 1024 bytes")
	}

	l = len(domain)
	if l < 1 || l > 1023 {
		return errors.New("domain must be between 1 and 1023 bytes")
	}
	return checkIP6String(domain)
}

func checkIP6String(domain string) error {
	// if the domain is a valid IPv6 address (with brackets), short circuit.
	if l := len(domain); l > 2 && strings.HasPrefix(domain, "[") &&
		strings.HasSuffix(domain, "]") {
		if ip := net.ParseIP(domain[1 : l-1]); ip == nil || ip.To4() != nil {
			return errors.New("domain is not a valid IPv6 address")
		}
	}
	return nil
}
