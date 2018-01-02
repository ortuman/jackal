/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
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
	Element
	to   *JID
	from *JID
}

type MutableMessage struct {
	MutableElement
}

func NewMessage(e *Element, from *JID, to *JID) (*Message, error) {
	if e.name != "message" {
		return nil, fmt.Errorf("wrong Message element name: %s", e.name)
	}
	messageType := e.Type()
	if !isMessageType(messageType) {
		return nil, fmt.Errorf(`invalid Message "type" attribute: %s`, messageType)
	}
	m := &Message{}
	m.name = e.name
	m.copyAttributes(e.attrs)
	m.copyElements(e.elements)
	m.setAttribute("to", to.ToFullJID())
	m.setAttribute("from", from.ToFullJID())
	m.to = to
	m.from = from
	return m, nil
}

func NewMutableMessage() *MutableMessage {
	m := &MutableMessage{}
	m.SetName("message")
	return m
}

func NewMutableMessageType(messageType string) *MutableMessage {
	m := &MutableMessage{}
	m.SetName("message")
	m.SetType(messageType)
	return m
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

func isMessageType(messageType string) bool {
	switch messageType {
	case "", NormalType, HeadlineType, ChatType, GroupChatType:
		return true
	default:
		return false
	}
}
