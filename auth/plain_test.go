/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/ortuman/jackal/model"
	memorystorage "github.com/ortuman/jackal/storage/memory"
	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

const (
	password       = "1234"
	iterationCount = 1
	salt           = "salt"
)

func TestAuthPlainAuthentication(t *testing.T) {
	var err error

	saltedPassword := SaltedPassword([]byte(password), []byte(salt), iterationCount, sha256.New)
	testStm, s := authTestSetup(&model.User{
		Username:            "mariana",
		PasswordScramSHA256: saltedPassword,
		IterationCount:      iterationCount,
		Salt:                []byte(salt),
	})

	authr := NewPlain(testStm, s)
	require.Equal(t, authr.Mechanism(), "PLAIN")
	require.False(t, authr.UsesChannelBinding())

	elem := xmpp.NewElementNamespace("auth", "urn:ietf:params:xml:ns:xmpp-sasl")
	elem.SetAttribute("mechanism", "PLAIN")
	_ = authr.ProcessElement(context.Background(), elem)

	buf := new(bytes.Buffer)
	buf.WriteByte(0)
	buf.WriteString("mariana")
	buf.WriteByte(0)
	buf.WriteString(password)
	elem.SetText(base64.StdEncoding.EncodeToString(buf.Bytes()))

	// storage error...
	memorystorage.EnableMockedError()
	require.Equal(t, authr.ProcessElement(context.Background(), elem), memorystorage.ErrMocked)
	memorystorage.DisableMockedError()

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
	buf.WriteString(password)
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
	buf.WriteString(password)
	elem.SetText(base64.StdEncoding.EncodeToString(buf.Bytes()))

	authr.Reset()
	err = authr.ProcessElement(context.Background(), elem)
	require.Equal(t, ErrSASLNotAuthorized, err)

	// incorrect password
	buf.Reset()
	buf.WriteByte(0)
	buf.WriteString("mariana")
	buf.WriteByte(0)
	buf.WriteString(password + "5")
	elem.SetText(base64.StdEncoding.EncodeToString(buf.Bytes()))

	authr.Reset()
	err = authr.ProcessElement(context.Background(), elem)
	require.Equal(t, ErrSASLNotAuthorized, err)
}
