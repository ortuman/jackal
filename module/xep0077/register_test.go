/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0077

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"testing"

	"github.com/ortuman/jackal/router/host"

	"github.com/ortuman/jackal/auth"
	c2srouter "github.com/ortuman/jackal/c2s/router"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/router"
	memorystorage "github.com/ortuman/jackal/storage/memory"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func newUser() *model.User {
	const (
		password       = "1234"
		iterationCount = 1
		salt           = "salt"
	)
	passwordScramSHA256 := auth.SaltedPassword([]byte(password), []byte(salt), iterationCount, sha256.New)
	return &model.User{
		Username:            "ortuman",
		PasswordScramSHA256: passwordScramSHA256,
		Salt:                []byte(salt),
		IterationCount:      iterationCount,
	}
}

func TestXEP0077_Matching(t *testing.T) {
	r, s := setupTest("jackal.im")

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	x := New(&Config{}, nil, r, s)
	defer func() { _ = x.Shutdown() }()

	// test MatchesIQ
	iq := xmpp.NewIQType(uuid.New(), xmpp.SetType)
	iq.SetFromJID(j)

	require.False(t, x.MatchesIQ(iq))
	iq.AppendElement(xmpp.NewElementNamespace("query", registerNamespace))
	require.True(t, x.MatchesIQ(iq))
}

func TestXEP0077_InvalidToJID(t *testing.T) {
	r, s := setupTest("jackal.im")

	j1, _ := jid.New("romeo", "jackal.im", "balcony", true)
	j2, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	r.Bind(context.Background(), stm1)

	x := New(&Config{}, nil, r, s)
	defer func() { _ = x.Shutdown() }()

	iq := xmpp.NewIQType(uuid.New(), xmpp.SetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j2.ToBareJID())
	stm1.SetAuthenticated(true)

	x.ProcessIQ(context.Background(), iq)
	elem := stm1.ReceiveElement()
	require.Equal(t, xmpp.ErrForbidden.Error(), elem.Error().Elements().All()[0].Name())

	iq2 := xmpp.NewIQType(uuid.New(), xmpp.SetType)
	iq2.SetFromJID(j1)
	iq2.SetToJID(j1.ToBareJID())
}

func TestXEP0077_NotAuthenticatedErrors(t *testing.T) {
	r, s := setupTest("jackal.im")

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	r.Bind(context.Background(), stm)

	x := New(&Config{}, nil, r, s)
	defer func() { _ = x.Shutdown() }()

	iq := xmpp.NewIQType(uuid.New(), xmpp.ResultType)
	iq.SetFromJID(j)
	iq.SetToJID(j.ToBareJID())

	x.ProcessIQ(context.Background(), iq)
	elem := stm.ReceiveElement()
	require.Equal(t, xmpp.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	iq.SetType(xmpp.GetType)
	x.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrNotAllowed.Error(), elem.Error().Elements().All()[0].Name())

	// allow registration...
	x = New(&Config{AllowRegistration: true}, nil, r, s)
	defer func() { _ = x.Shutdown() }()

	q := xmpp.NewElementNamespace("query", registerNamespace)
	q.AppendElement(xmpp.NewElementName("q2"))
	iq.AppendElement(q)

	x.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	q.ClearElements()
	iq.SetType(xmpp.SetType)
	stm.SetValue(xep077RegisteredCtxKey, true)

	x.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrNotAcceptable.Error(), elem.Error().Elements().All()[0].Name())
}

func TestXEP0077_AuthenticatedErrors(t *testing.T) {
	r, s := setupTest("jackal.im")

	srvJid, _ := jid.New("", "jackal.im", "", true)
	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	r.Bind(context.Background(), stm)

	stm.SetAuthenticated(true)

	x := New(&Config{}, nil, r, s)
	defer func() { _ = x.Shutdown() }()

	iq := xmpp.NewIQType(uuid.New(), xmpp.ResultType)
	iq.SetFromJID(j)
	iq.SetToJID(j.ToBareJID())
	iq.SetToJID(srvJid)

	x.ProcessIQ(context.Background(), iq)
	elem := stm.ReceiveElement()
	require.Equal(t, xmpp.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	iq.SetType(xmpp.SetType)
	iq.AppendElement(xmpp.NewElementNamespace("query", registerNamespace))
	x.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())
}

func TestXEP0077_RegisterUser(t *testing.T) {
	r, s := setupTest("jackal.im")

	srvJid, _ := jid.New("", "jackal.im", "", true)
	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	r.Bind(context.Background(), stm)

	x := New(&Config{AllowRegistration: true}, nil, r, s)
	defer func() { _ = x.Shutdown() }()

	iq := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq.SetFromJID(j)
	iq.SetToJID(srvJid)

	q := xmpp.NewElementNamespace("query", registerNamespace)
	iq.AppendElement(q)

	x.ProcessIQ(context.Background(), iq)
	q2 := stm.ReceiveElement().Elements().ChildNamespace("query", registerNamespace)
	require.NotNil(t, q2.Elements().Child("username"))
	require.NotNil(t, q2.Elements().Child("password"))

	username := xmpp.NewElementName("username")
	password := xmpp.NewElementName("password")
	q.AppendElement(username)
	q.AppendElement(password)

	// empty fields
	iq.SetType(xmpp.SetType)
	x.ProcessIQ(context.Background(), iq)
	elem := stm.ReceiveElement()
	require.Equal(t, xmpp.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	// already existing user...
	_ = s.UpsertUser(context.Background(), newUser())
	username.SetText("ortuman")
	password.SetText("5678")
	x.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrConflict.Error(), elem.Error().Elements().All()[0].Name())

	// storage error
	memorystorage.EnableMockedError()
	x.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	memorystorage.DisableMockedError()

	username.SetText("juliet")
	x.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ResultType, elem.Type())

	usr, _ := s.FetchUser(context.Background(), "ortuman")
	require.NotNil(t, usr)
}

func TestXEP0077_CancelRegistration(t *testing.T) {
	r, s := setupTest("jackal.im")

	srvJid, _ := jid.New("", "jackal.im", "", true)
	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S("abcd1234", j)
	r.Bind(context.Background(), stm)

	stm.SetAuthenticated(true)

	x := New(&Config{}, nil, r, s)
	defer func() { _ = x.Shutdown() }()

	_ = s.UpsertUser(context.Background(), newUser())

	iq := xmpp.NewIQType(uuid.New(), xmpp.SetType)
	iq.SetFromJID(j)
	iq.SetToJID(srvJid)

	q := xmpp.NewElementNamespace("query", registerNamespace)
	q.AppendElement(xmpp.NewElementName("remove"))

	iq.AppendElement(q)
	x.ProcessIQ(context.Background(), iq)
	elem := stm.ReceiveElement()
	require.Equal(t, xmpp.ErrNotAllowed.Error(), elem.Error().Elements().All()[0].Name())

	x = New(&Config{AllowCancel: true}, nil, r, s)
	defer func() { _ = x.Shutdown() }()

	q.AppendElement(xmpp.NewElementName("remove2"))
	x.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())
	q.ClearElements()
	q.AppendElement(xmpp.NewElementName("remove"))

	// storage error
	memorystorage.EnableMockedError()
	x.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	memorystorage.DisableMockedError()

	x.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ResultType, elem.Type())

	usr, _ := s.FetchUser(context.Background(), "ortuman")
	require.Nil(t, usr)
}

func TestXEP0077_ChangePassword(t *testing.T) {
	r, s := setupTest("jackal.im")

	srvJid, _ := jid.New("", "jackal.im", "", true)
	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	r.Bind(context.Background(), stm)

	stm.SetAuthenticated(true)

	x := New(&Config{}, nil, r, s)
	defer func() { _ = x.Shutdown() }()

	_ = s.UpsertUser(context.Background(), newUser())

	iq := xmpp.NewIQType(uuid.New(), xmpp.SetType)
	iq.SetFromJID(j)
	iq.SetToJID(srvJid)

	q := xmpp.NewElementNamespace("query", registerNamespace)
	username := xmpp.NewElementName("username")
	username.SetText("juliet")
	password := xmpp.NewElementName("password")
	password.SetText("5678")
	q.AppendElement(username)
	q.AppendElement(password)
	iq.AppendElement(q)

	x.ProcessIQ(context.Background(), iq)
	elem := stm.ReceiveElement()
	require.Equal(t, xmpp.ErrNotAllowed.Error(), elem.Error().Elements().All()[0].Name())

	x = New(&Config{AllowChange: true}, nil, r, s)
	defer func() { _ = x.Shutdown() }()

	x.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrNotAllowed.Error(), elem.Error().Elements().All()[0].Name())

	username.SetText("ortuman")
	x.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrNotAuthorized.Error(), elem.Error().Elements().All()[0].Name())

	// secure channel...
	stm.SetSecured(true)

	// storage error
	memorystorage.EnableMockedError()
	x.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	memorystorage.DisableMockedError()

	x.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ResultType, elem.Type())

	usr, _ := s.FetchUser(context.Background(), "ortuman")
	require.NotNil(t, usr)

	passwordScramSHA256 := auth.SaltedPassword([]byte("5678"), usr.Salt, usr.IterationCount, sha256.New)
	require.Equal(t, passwordScramSHA256, usr.PasswordScramSHA256)
}

func setupTest(domain string) (router.Router, *memorystorage.User) {
	hosts, _ := host.New([]host.Config{{Name: domain, Certificate: tls.Certificate{}}})
	userRep := memorystorage.NewUser()
	r, _ := router.New(
		hosts,
		c2srouter.New(userRep, memorystorage.NewBlockList()),
		nil,
	)
	return r, userRep
}
