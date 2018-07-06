/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package auth

import (
	"testing"

	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xml/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func authTestSetup(user *model.User) *stream.MockC2S {
	storage.Initialize(&storage.Config{Type: storage.Memory})

	storage.Instance().InsertOrUpdateUser(user)

	j, _ := jid.New("mariana", "localhost", "res", true)

	testStrm := stream.NewMockC2S(uuid.New(), j)
	testStrm.SetUsername("mariana")
	testStrm.SetDomain("localhost")
	testStrm.SetResource("res")

	testStrm.SetJID(j)
	return testStrm
}

func authTestTeardown() {
	storage.Instance().Shutdown()
}

func TestAuthError(t *testing.T) {
	require.Equal(t, "incorrect-encoding", ErrSASLIncorrectEncoding.(*SASLError).Error())
	require.Equal(t, "malformed-request", ErrSASLMalformedRequest.(*SASLError).Error())
	require.Equal(t, "not-authorized", ErrSASLNotAuthorized.(*SASLError).Error())
	require.Equal(t, "temporary-auth-failure", ErrSASLTemporaryAuthFailure.(*SASLError).Error())

	require.Equal(t, "incorrect-encoding", ErrSASLIncorrectEncoding.(*SASLError).Element().Name())
	require.Equal(t, "malformed-request", ErrSASLMalformedRequest.(*SASLError).Element().Name())
	require.Equal(t, "not-authorized", ErrSASLNotAuthorized.(*SASLError).Element().Name())
	require.Equal(t, "temporary-auth-failure", ErrSASLTemporaryAuthFailure.(*SASLError).Element().Name())
}
