/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import "fmt"

const (
	NormalType    = "normal"
	HeadlineType  = "headline"
	ChatType      = "chat"
	GroupChatType = "groupchat"
)

type Message struct {
	XElement
	to   *JID
	from *JID
}

func NewMessageFromElement(e Element, from *JID, to *JID) (*Message, error) {
	if e.Name() != "message" {
		return nil, fmt.Errorf("wrong Message element name: %s", e.Name())
	}
	messageType := e.Type()
	if !isMessageType(messageType) {
		return nil, fmt.Errorf(`invalid Message "type" attribute: %s`, messageType)
	}
	m := &Message{}
	m.name = e.Name()
	m.attrs = e.Attributes()
	m.elements = e.Elements()
	m.SetAttribute("to", to.String())
	m.SetAttribute("from", from.String())
	m.to = to
	m.from = from
	return m, nil
}

// IsNormal returns true if this is a 'normal' type Message.
func (m *Message) IsNormal() bool {
	return m.Type() == NormalType || m.Type() == ""
}

// IsHeadline returns true if this is a 'headline' type Message.
func (m *Message) IsHeadline() bool {
	return m.Type() == HeadlineType
}

// IsChat returns true if this is a 'chat' type Message.
func (m *Message) IsChat() bool {
	return m.Type() == ChatType
}

// IsGroupChat returns true if this is a 'groupchat' type Message.
func (m *Message) IsGroupChat() bool {
	return m.Type() == GroupChatType
}

// ToJID satisfies stanza interface.
func (m *Message) ToJID() *JID {
	return m.to
}

// FromJID satisfies stanza interface.
func (m *Message) FromJID() *JID {
	return m.from
}

// Copy returns a deep copy of this message stanza.
func (m *Message) Copy() *Message {
	cp := &Message{}
	cp.name = m.name
	cp.text = m.text
	cp.attrs = m.attrs
	cp.elements = m.elements
	return cp
}

func isMessageType(messageType string) bool {
	switch messageType {
	case "", NormalType, HeadlineType, ChatType, GroupChatType:
		return true
	default:
		return false
	}
}
