/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package auth

import (
	"testing"

	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/memstorage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func authTestSetup(user *model.User) (*stream.MockC2S, *memstorage.Storage) {
	s := memstorage.New()
	storage.Set(s)

	_ = storage.UpsertUser(user)

	j, _ := jid.New("mariana", "localhost", "res", true)

	testStm := stream.NewMockC2S(uuid.New(), j)

	testStm.SetJID(j)
	return testStm, s
}

func authTestTeardown() {
	storage.Unset()
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
