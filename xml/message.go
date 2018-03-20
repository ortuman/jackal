/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import (
	"fmt"
)

const (
	// NormalType represents a 'normal' message type.
	NormalType = "normal"

	// HeadlineType represents a 'headline' message type.
	HeadlineType = "headline"

	// ChatType represents a 'chat' message type.
	ChatType = "chat"

	// GroupChatType represents a 'groupchat' message type.
	GroupChatType = "groupchat"
)

// Message type represents a <message> element.
// All incoming <message> elements providing from the
// stream will automatically be converted to Message objects.
type Message struct {
	MutableElement
	to   *JID
	from *JID
}

// NewMessageFromElement creates a Message object from Element.
func NewMessageFromElement(e Element, from *JID, to *JID) (*Message, error) {
	if e.Name() != "message" {
		return nil, fmt.Errorf("wrong Message element name: %s", e.Name())
	}
	messageType := e.Type()
	if !isMessageType(messageType) {
		return nil, fmt.Errorf(`invalid Message "type" attribute: %s`, messageType)
	}
	m := &Message{}
	m.copyFrom(e)
	m.SetToJID(to)
	m.SetFromJID(from)
	return m, nil
}

// NewMessageType creates and returns a new Message element.
func NewMessageType(identifier string, messageType string) *Message {
	msg := &Message{}
	msg.SetName("message")
	msg.SetID(identifier)
	msg.SetType(messageType)
	return msg
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

// IsMessageWithBody returns true if the message
// has a body sub element.
func (m *Message) IsMessageWithBody() bool {
	return m.FindElement("body") != nil
}

// ToJID returns message 'to' JID value.
func (m *Message) ToJID() *JID {
	return m.to
}

// SetToJID sets the message 'to' JID value.
func (m *Message) SetToJID(to *JID) {
	m.to = to
	m.SetAttribute("to", to.String())
}

// FromJID returns message 'from' JID value.
func (m *Message) FromJID() *JID {
	return m.from
}

// SetFromJID sets the message 'from' JID value.
func (m *Message) SetFromJID(from *JID) {
	m.from = from
	m.SetAttribute("from", from.String())
}

func isMessageType(messageType string) bool {
	switch messageType {
	case "", ErrorType, NormalType, HeadlineType, ChatType, GroupChatType:
		return true
	default:
		return false
	}
}
