/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"

	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/storage/memstorage"
	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestAuthPlainAuthentication(t *testing.T) {
	var err error

	testStm, s := authTestSetup(&model.User{Username: "mariana", Password: "1234"})
	defer authTestTeardown()

	authr := NewPlain(testStm)
	require.Equal(t, authr.Mechanism(), "PLAIN")
	require.False(t, authr.UsesChannelBinding())

	elem := xmpp.NewElementNamespace("auth", "urn:ietf:params:xml:ns:xmpp-sasl")
	elem.SetAttribute("mechanism", "PLAIN")
	authr.ProcessElement(context.Background(), elem)

	buf := new(bytes.Buffer)
	buf.WriteByte(0)
	buf.WriteString("mariana")
	buf.WriteByte(0)
	buf.WriteString("1234")
	elem.SetText(base64.StdEncoding.EncodeToString(buf.Bytes()))

	// storage error...
	s.EnableMockedError()
	require.Equal(t, authr.ProcessElement(context.Background(), elem), memstorage.ErrMockedError)
	s.DisableMockedError()

	// valid credentials...
	err = authr.ProcessElement(context.Background(), elem)
	require.Nil(t, err)
	require.Equal(t, "mariana", authr.Username())
	require.True(t, authr.Authenticated())

	// already authenticated...
	err = authr.ProcessElement(context.Background(), elem)
	require.Nil(t, err)

	// malformed request
	authr.Reset()
	elem.SetText("")
	err = authr.ProcessElement(context.Background(), elem)
	require.Equal(t, ErrSASLMalformedRequest, err)

	// invalid payload
	authr.Reset()
	elem.SetText("bad formed base64")
	err = authr.ProcessElement(context.Background(), elem)
	require.Equal(t, ErrSASLIncorrectEncoding, err)

	// invalid payload
	buf.Reset()
	buf.WriteByte(0)
	buf.WriteString("mariana")
	buf.WriteByte(0)
	buf.WriteString("1234")
	buf.WriteByte(0)
	elem.SetText(base64.StdEncoding.EncodeToString(buf.Bytes()))

	authr.Reset()
	err = authr.ProcessElement(context.Background(), elem)
	require.Equal(t, ErrSASLIncorrectEncoding, err)

	// invalid user
	buf.Reset()
	buf.WriteByte(0)
	buf.WriteString("ortuman")
	buf.WriteByte(0)
	buf.WriteString("1234")
	elem.SetText(base64.StdEncoding.EncodeToString(buf.Bytes()))

	authr.Reset()
	err = authr.ProcessElement(context.Background(), elem)
	require.Equal(t, ErrSASLNotAuthorized, err)

	// incorrect password
	buf.Reset()
	buf.WriteByte(0)
	buf.WriteString("mariana")
	buf.WriteByte(0)
	buf.WriteString("12345")
	elem.SetText(base64.StdEncoding.EncodeToString(buf.Bytes()))

	authr.Reset()
	err = authr.ProcessElement(context.Background(), elem)
	require.Equal(t, ErrSASLNotAuthorized, err)
}
