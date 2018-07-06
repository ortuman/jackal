/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0077

import (
	"testing"

	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xml"
	"github.com/ortuman/jackal/xml/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0077_Matching(t *testing.T) {
	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	x := New(&Config{}, nil)

	// test MatchesIQ
	iq := xml.NewIQType(uuid.New(), xml.SetType)
	iq.SetFromJID(j)

	require.False(t, x.MatchesIQ(iq))
	iq.AppendElement(xml.NewElementNamespace("query", registerNamespace))
	require.True(t, x.MatchesIQ(iq))
}

func TestXEP0077_InvalidToJID(t *testing.T) {
	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S("abcd1234", j)
	defer stm.Disconnect(nil)

	x := New(&Config{}, stm)

	stm.SetUsername("romeo")
	iq := xml.NewIQType(uuid.New(), xml.SetType)
	iq.SetFromJID(j)
	iq.SetToJID(j.ToBareJID())

	x.ProcessIQ(iq)
	elem := stm.FetchElement()
	require.Equal(t, xml.ErrForbidden.Error(), elem.Error().Elements().All()[0].Name())

	iq2 := xml.NewIQType(uuid.New(), xml.SetType)
	iq2.SetFromJID(j)
	iq2.SetToJID(j.ToBareJID())

	stm.SetUsername("ortuman")
	stm.SetAuthenticated(true)
	x.ProcessIQ(iq2)
	elem = stm.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.ErrForbidden.Error(), elem.Error().Elements().All()[0].Name())
}

func TestXEP0077_NotAuthenticatedErrors(t *testing.T) {
	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S("abcd1234", j)
	defer stm.Disconnect(nil)

	x := New(&Config{}, stm)

	iq := xml.NewIQType(uuid.New(), xml.ResultType)
	iq.SetFromJID(j)
	iq.SetToJID(j.ToBareJID())

	x.ProcessIQ(iq)
	elem := stm.FetchElement()
	require.Equal(t, xml.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	iq.SetType(xml.GetType)
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrNotAllowed.Error(), elem.Error().Elements().All()[0].Name())

	// allow registration...
	x = New(&Config{AllowRegistration: true}, stm)

	q := xml.NewElementNamespace("query", registerNamespace)
	q.AppendElement(xml.NewElementName("q2"))
	iq.AppendElement(q)

	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	q.ClearElements()
	iq.SetType(xml.SetType)
	x.registered = true

	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrNotAcceptable.Error(), elem.Error().Elements().All()[0].Name())
}

func TestXEP0077_AuthenticatedErrors(t *testing.T) {
	srvJid, _ := jid.New("", "jackal.im", "", true)
	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S("abcd1234", j)
	defer stm.Disconnect(nil)

	stm.SetAuthenticated(true)

	x := New(&Config{}, stm)

	iq := xml.NewIQType(uuid.New(), xml.ResultType)
	iq.SetFromJID(j)
	iq.SetToJID(j.ToBareJID())
	iq.SetToJID(srvJid)

	x.ProcessIQ(iq)
	elem := stm.FetchElement()
	require.Equal(t, xml.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	iq.SetType(xml.SetType)
	iq.AppendElement(xml.NewElementNamespace("query", registerNamespace))
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())
}

func TestXEP0077_RegisterUser(t *testing.T) {
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer storage.Shutdown()

	srvJid, _ := jid.New("", "jackal.im", "", true)
	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S("abcd1234", j)
	defer stm.Disconnect(nil)

	x := New(&Config{AllowRegistration: true}, stm)

	iq := xml.NewIQType(uuid.New(), xml.GetType)
	iq.SetFromJID(srvJid)
	iq.SetToJID(srvJid)

	q := xml.NewElementNamespace("query", registerNamespace)
	iq.AppendElement(q)

	x.ProcessIQ(iq)
	q2 := stm.FetchElement().Elements().ChildNamespace("query", registerNamespace)
	require.NotNil(t, q2.Elements().Child("username"))
	require.NotNil(t, q2.Elements().Child("password"))

	username := xml.NewElementName("username")
	password := xml.NewElementName("password")
	q.AppendElement(username)
	q.AppendElement(password)

	// empty fields
	iq.SetType(xml.SetType)
	x.ProcessIQ(iq)
	elem := stm.FetchElement()
	require.Equal(t, xml.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	// already existing user...
	storage.Instance().InsertOrUpdateUser(&model.User{Username: "ortuman", Password: "1234"})
	username.SetText("ortuman")
	password.SetText("5678")
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrConflict.Error(), elem.Error().Elements().All()[0].Name())

	// storage error
	storage.ActivateMockedError()
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())

	storage.DeactivateMockedError()
	username.SetText("juliet")
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ResultType, elem.Type())

	usr, _ := storage.Instance().FetchUser("ortuman")
	require.NotNil(t, usr)
}

func TestXEP0077_CancelRegistration(t *testing.T) {
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer storage.Shutdown()

	srvJid, _ := jid.New("", "jackal.im", "", true)
	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S("abcd1234", j)
	defer stm.Disconnect(nil)

	stm.SetAuthenticated(true)

	x := New(&Config{}, stm)

	storage.Instance().InsertOrUpdateUser(&model.User{Username: "ortuman", Password: "1234"})

	iq := xml.NewIQType(uuid.New(), xml.SetType)
	iq.SetFromJID(srvJid)
	iq.SetToJID(srvJid)

	q := xml.NewElementNamespace("query", registerNamespace)
	q.AppendElement(xml.NewElementName("remove"))

	iq.AppendElement(q)
	x.ProcessIQ(iq)
	elem := stm.FetchElement()
	require.Equal(t, xml.ErrNotAllowed.Error(), elem.Error().Elements().All()[0].Name())

	x = New(&Config{AllowCancel: true}, stm)

	q.AppendElement(xml.NewElementName("remove2"))
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())
	q.ClearElements()
	q.AppendElement(xml.NewElementName("remove"))

	// storage error
	storage.ActivateMockedError()
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	storage.DeactivateMockedError()

	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ResultType, elem.Type())

	usr, _ := storage.Instance().FetchUser("ortuman")
	require.Nil(t, usr)
}

func TestXEP0077_ChangePassword(t *testing.T) {
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer storage.Shutdown()

	srvJid, _ := jid.New("", "jackal.im", "", true)
	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S("abcd1234", j)
	defer stm.Disconnect(nil)

	stm.SetAuthenticated(true)

	x := New(&Config{}, stm)

	storage.Instance().InsertOrUpdateUser(&model.User{Username: "ortuman", Password: "1234"})

	iq := xml.NewIQType(uuid.New(), xml.SetType)
	iq.SetFromJID(srvJid)
	iq.SetToJID(srvJid)

	q := xml.NewElementNamespace("query", registerNamespace)
	username := xml.NewElementName("username")
	username.SetText("juliet")
	password := xml.NewElementName("password")
	password.SetText("5678")
	q.AppendElement(username)
	q.AppendElement(password)
	iq.AppendElement(q)

	x.ProcessIQ(iq)
	elem := stm.FetchElement()
	require.Equal(t, xml.ErrNotAllowed.Error(), elem.Error().Elements().All()[0].Name())

	x = New(&Config{AllowChange: true}, stm)

	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrNotAllowed.Error(), elem.Error().Elements().All()[0].Name())

	username.SetText("ortuman")
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrNotAuthorized.Error(), elem.Error().Elements().All()[0].Name())

	// secure channel...
	stm.SetSecured(true)

	// storage error
	storage.ActivateMockedError()
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	storage.DeactivateMockedError()

	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ResultType, elem.Type())

	usr, _ := storage.Instance().FetchUser("ortuman")
	require.NotNil(t, usr)
	require.Equal(t, "5678", usr.Password)
}
