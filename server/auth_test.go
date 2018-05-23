/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package server

import (
	"testing"

	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func authTestSetup(user *model.User) *router.MockC2S {
	storage.Initialize(&storage.Config{Type: storage.Memory})

	storage.Instance().InsertOrUpdateUser(user)

	jid, _ := xml.NewJID("mariana", "localhost", "res", true)

	testStrm := router.NewMockC2S(uuid.New(), jid)
	testStrm.SetUsername("mariana")
	testStrm.SetDomain("localhost")
	testStrm.SetResource("res")

	testStrm.SetJID(jid)
	return testStrm
}

func authTestTeardown() {
	storage.Instance().Shutdown()
}

func TestAuthError(t *testing.T) {
	require.Equal(t, "incorrect-encoding", errSASLIncorrectEncoding.(*saslErrorString).Error())
	require.Equal(t, "malformed-request", errSASLMalformedRequest.(*saslErrorString).Error())
	require.Equal(t, "not-authorized", errSASLNotAuthorized.(*saslErrorString).Error())
	require.Equal(t, "temporary-auth-failure", errSASLTemporaryAuthFailure.(*saslErrorString).Error())

	require.Equal(t, "incorrect-encoding", errSASLIncorrectEncoding.(*saslErrorString).Element().Name())
	require.Equal(t, "malformed-request", errSASLMalformedRequest.(*saslErrorString).Element().Name())
	require.Equal(t, "not-authorized", errSASLNotAuthorized.(*saslErrorString).Element().Name())
	require.Equal(t, "temporary-auth-failure", errSASLTemporaryAuthFailure.(*saslErrorString).Element().Name())
}
