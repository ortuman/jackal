// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"hash"
	"strconv"
	"strings"
	"testing"

	"github.com/jackal-xmpp/stravaganza"
	"github.com/ortuman/jackal/pkg/auth/pepper"
	usermodel "github.com/ortuman/jackal/pkg/model/user"
	"github.com/ortuman/jackal/pkg/transport"
	stringsutil "github.com/ortuman/jackal/pkg/util/strings"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/sha3"
)

const (
	tSaltBase64     = "IqUdyUqUslbGOH5p8tCkev0fp7IiTXjgjO3_0SgPjrI"
	pepperKey       = "6ZavKvaLqSFGM5zDnFq7WWih"
	tIterationCount = 100_000

	tScramSHA1Base64    = "aCX4c9PCk4bmwAkmWKyjtcfd1Pg"
	tScramSHA256Base64  = "vXILnAD7G9dZ4k8XWezBixFs-vt2qL642R6xa1sPIFI"
	tScramSHA512Base64  = "_VfygyYVA_wKJmvtGWobubcGdwVfghOohXVVeyDqW6ra26SQFxRwStdctM-dSC7TW7oF_GHdCTK0HzGQLqMx3A"
	tScramSHA3512Base64 = "N6tUI6iQTEjyPg86NGfYbzp0K8KVbB0tUOfFe9ZDwIh8ssF6qcTx5G8UFmotrWnOky6TJTQprjmOn66ffK1tZw"
)

var scramTypeStr = map[ScramType]string{
	ScramSHA1:    "ScramSHA-1",
	ScramSHA256:  "ScramSHA-256",
	ScramSHA512:  "ScramSHA-512",
	ScramSHA3512: "ScramSHA3-512",
}

type scramAuthTestCase struct {
	name              string
	scramType         ScramType
	usesCb            bool
	cbBytes           []byte
	gs2BindFlag       string
	authID            string
	n                 string
	r                 string
	password          string
	expectsError      bool
	expectedErrReason SASLErrorReason
}

type scramAuthResult struct {
	clientFinalMessage string
	v                  string
}

func TestScram_Mechanisms(t *testing.T) {
	// given
	auth0 := &Scram{tp: ScramSHA1, usesCb: false}
	auth1 := &Scram{tp: ScramSHA1, usesCb: true}

	auth2 := &Scram{tp: ScramSHA256, usesCb: false}
	auth3 := &Scram{tp: ScramSHA256, usesCb: true}

	auth4 := &Scram{tp: ScramSHA512, usesCb: false}
	auth5 := &Scram{tp: ScramSHA512, usesCb: true}

	auth6 := &Scram{tp: ScramSHA3512, usesCb: false}
	auth7 := &Scram{tp: ScramSHA3512, usesCb: true}

	// then
	require.Equal(t, auth0.Mechanism(), "SCRAM-SHA-1")
	require.False(t, auth0.UsesChannelBinding())

	require.Equal(t, auth1.Mechanism(), "SCRAM-SHA-1-PLUS")
	require.True(t, auth1.UsesChannelBinding())

	require.Equal(t, auth2.Mechanism(), "SCRAM-SHA-256")
	require.False(t, auth2.UsesChannelBinding())

	require.Equal(t, auth3.Mechanism(), "SCRAM-SHA-256-PLUS")
	require.True(t, auth3.UsesChannelBinding())

	require.Equal(t, auth4.Mechanism(), "SCRAM-SHA-512")
	require.False(t, auth4.UsesChannelBinding())

	require.Equal(t, auth5.Mechanism(), "SCRAM-SHA-512-PLUS")
	require.True(t, auth5.UsesChannelBinding())

	require.Equal(t, auth6.Mechanism(), "SCRAM-SHA3-512")
	require.False(t, auth6.UsesChannelBinding())

	require.Equal(t, auth7.Mechanism(), "SCRAM-SHA3-512-PLUS")
	require.True(t, auth7.UsesChannelBinding())
}

func TestScram_Cases(t *testing.T) {
	var tps = []ScramType{ScramSHA1, ScramSHA256, ScramSHA512, ScramSHA3512}
	for _, tp := range tps {
		testScramTypeCases(t, tp)
	}
}

func testScramTypeCases(t *testing.T, tp ScramType) {
	var tcs = []scramAuthTestCase{
		{
			// Success
			name:        "Success",
			scramType:   tp,
			usesCb:      false,
			gs2BindFlag: "n",
			n:           "ortuman",
			r:           "bb769406-eaa4-4f38-a279-2b90e596f6dd",
			password:    "1234",
		},
		{
			// Success (PLUS)
			name:        "SuccessPLUS",
			scramType:   tp,
			usesCb:      true,
			cbBytes:     randomBytes(23),
			gs2BindFlag: "p=tls-unique",
			authID:      "a=jackal.im",
			n:           "ortuman",
			r:           "7e51aff7-6875-4dce-820a-6d4970635006",
			password:    "1234",
		},
		{
			// Invalid user
			name:              "InvalidUser",
			scramType:         tp,
			usesCb:            false,
			gs2BindFlag:       "n",
			n:                 "mariana",
			r:                 "bb769406-eaa4-4f38-a279-2b90e596f6dd",
			password:          "1234",
			expectsError:      true,
			expectedErrReason: NotAuthorized,
		},
		{
			// Invalid password
			name:              "InvalidPassword",
			scramType:         tp,
			usesCb:            false,
			gs2BindFlag:       "n",
			n:                 "ortuman",
			r:                 "bb769406-eaa4-4f38-a279-2b90e596f6dd",
			password:          "12345678",
			expectsError:      true,
			expectedErrReason: NotAuthorized,
		},
		{
			// No matching gs2BindFlag
			name:              "NoMatchingG2SBindFlag",
			scramType:         tp,
			usesCb:            false,
			gs2BindFlag:       "p=tls-unique",
			authID:            "a=jackal.im",
			n:                 "ortuman",
			r:                 "bb769406-eaa4-4f38-a279-2b90e596f6dd",
			password:          "1234",
			expectsError:      true,
			expectedErrReason: NotAuthorized,
		},
		{
			// No matching gs2BindFlag (malformed)
			name:              "NoMatchingG2SBindFlagMalformed",
			scramType:         tp,
			usesCb:            false,
			gs2BindFlag:       "q=tls-unique",
			authID:            "a=jackal.im",
			n:                 "ortuman",
			r:                 "bb769406-eaa4-4f38-a279-2b90e596f6dd",
			password:          "1234",
			expectsError:      true,
			expectedErrReason: MalformedRequest,
		},
		{
			// Invalid authID
			name:              "InvalidAuthID",
			scramType:         tp,
			usesCb:            false,
			gs2BindFlag:       "n",
			authID:            "b=jackal.im",
			n:                 "ortuman",
			r:                 "bb769406-eaa4-4f38-a279-2b90e596f6dd",
			password:          "1234",
			expectsError:      true,
			expectedErrReason: MalformedRequest,
		},
		{
			// Empty username
			name:              "EmptyUsername",
			scramType:         tp,
			usesCb:            false,
			gs2BindFlag:       "n",
			authID:            "a=jackal.im",
			n:                 "",
			r:                 "bb769406-eaa4-4f38-a279-2b90e596f6dd",
			password:          "1234",
			expectsError:      true,
			expectedErrReason: MalformedRequest,
		},
	}
	for _, tc := range tcs {
		t.Run(fmt.Sprintf("%s/%s", scramTypeStr[tp], tc.name), func(t *testing.T) {
			if saslErr := processScramTestCase(t, &tc); saslErr != nil {
				if !tc.expectsError {
					require.Fail(t, fmt.Sprintf("Unexpected SASL error with reason: %d", tc.expectedErrReason))
				}
				require.Equal(t, tc.expectedErrReason, saslErr.Reason)
			} else if tc.expectsError {
				require.Fail(t, fmt.Sprintf("Expecting SASL error with reason: %d", tc.expectedErrReason))
			}
		})
	}
}

func processScramTestCase(t *testing.T, tc *scramAuthTestCase) *SASLError {
	trMock := &transportMock{}
	repMock := &usersRepository{}

	trMock.ChannelBindingBytesFunc = func(_ transport.ChannelBindingMechanism) []byte {
		return tc.cbBytes
	}
	testUsr := testUser()
	repMock.FetchUserFunc = func(_ context.Context, username string) (*usermodel.User, error) {
		if username != "ortuman" {
			return nil, nil
		}
		return testUsr, nil
	}
	auth := NewScram(trMock, tc.scramType, tc.usesCb, repMock, testPeppers())

	clientInitialMessage := fmt.Sprintf(`n=%s,r=%s`, tc.n, tc.r)
	gs2Header := fmt.Sprintf(`%s,%s,`, tc.gs2BindFlag, tc.authID)
	authPayload := gs2Header + clientInitialMessage

	authElem := stravaganza.NewBuilder("auth").
		WithAttribute(stravaganza.Namespace, saslNamespace).
		WithAttribute("mechanism", auth.Mechanism()).
		WithText(base64.StdEncoding.EncodeToString([]byte(authPayload))).
		Build()

	challengeElem, saslErr := auth.ProcessElement(context.Background(), authElem)
	if saslErr != nil {
		return saslErr
	}
	require.NotNil(t, challengeElem)
	require.Equal(t, "challenge", challengeElem.Name())

	srvInitialMessage, err := base64.StdEncoding.DecodeString(challengeElem.Text())
	require.Nil(t, err)
	resp, err := parseScramResponse(challengeElem.Text())
	require.Nil(t, err)

	srvNonce := resp["r"]
	salt, err := base64.StdEncoding.DecodeString(resp["s"])
	require.Nil(t, err)

	iterations, _ := strconv.Atoi(resp["i"])

	buf := new(bytes.Buffer)
	buf.Write([]byte(gs2Header))
	if tc.usesCb {
		buf.Write(tc.cbBytes)
	}
	cBytes := base64.StdEncoding.EncodeToString(buf.Bytes())

	res := computeScramAuthResult(tc.scramType, clientInitialMessage, string(srvInitialMessage), srvNonce, cBytes, tc.password, salt, iterations)

	responseElem := stravaganza.NewBuilder("response").
		WithAttribute(stravaganza.Namespace, saslNamespace).
		WithText(base64.StdEncoding.EncodeToString([]byte(res.clientFinalMessage))).
		Build()

	successElem, saslErr := auth.ProcessElement(context.Background(), responseElem)
	if saslErr != nil {
		return saslErr
	}
	require.Equal(t, "success", successElem.Name())

	vb64, err := base64.StdEncoding.DecodeString(successElem.Text())
	require.Nil(t, err)
	require.Equal(t, res.v, string(vb64))

	require.True(t, auth.Authenticated())
	require.Equal(t, tc.n, auth.Username())

	return nil
}

func parseScramResponse(b64 string) (map[string]string, error) {
	s, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}
	ret := map[string]string{}
	s1 := strings.Split(string(s), ",")
	for _, s0 := range s1 {
		k, v := stringsutil.SplitKeyAndValue(s0, '=')
		ret[k] = v
	}
	return ret, nil
}

func computeScramAuthResult(scramType ScramType, clientInitialMessage, serverInitialMessage, srvNonce, cBytes, password string, salt []byte, iterations int) *scramAuthResult {
	clientFinalMessageBare := fmt.Sprintf("c=%s,r=%s", cBytes, srvNonce)

	saltedPassword := testScramAuthPbkdf2([]byte(password), salt, scramType, iterations)

	clientKey := testScramAuthHmac([]byte("Client Key"), saltedPassword, scramType)
	storedKey := testScramAuthHash(clientKey, scramType)
	authMessage := clientInitialMessage + "," + serverInitialMessage + "," + clientFinalMessageBare
	clientSignature := testScramAuthHmac([]byte(authMessage), storedKey, scramType)

	clientProof := make([]byte, len(clientKey))
	for i := 0; i < len(clientKey); i++ {
		clientProof[i] = clientKey[i] ^ clientSignature[i]
	}
	serverKey := testScramAuthHmac([]byte("Server Key"), saltedPassword, scramType)
	serverSignature := testScramAuthHmac([]byte(authMessage), serverKey, scramType)

	res := &scramAuthResult{}
	res.clientFinalMessage = clientFinalMessageBare + ",p=" + base64.StdEncoding.EncodeToString(clientProof)
	res.v = "v=" + base64.StdEncoding.EncodeToString(serverSignature)
	return res
}

func testScramAuthHmac(b []byte, key []byte, scramType ScramType) []byte {
	var h func() hash.Hash
	switch scramType {
	case ScramSHA1:
		h = sha1.New
	case ScramSHA256:
		h = sha256.New
	case ScramSHA512:
		h = sha512.New
	case ScramSHA3512:
		h = sha3.New512
	}
	m := hmac.New(h, key)
	m.Write(b)
	return m.Sum(nil)
}

func testScramAuthPbkdf2(b []byte, salt []byte, scramType ScramType, iterationCount int) []byte {
	switch scramType {
	case ScramSHA1:
		return pbkdf2.Key(b, salt, iterationCount, sha1.Size, sha1.New)
	case ScramSHA256:
		return pbkdf2.Key(b, salt, iterationCount, sha256.Size, sha256.New)
	case ScramSHA512:
		return pbkdf2.Key(b, salt, iterationCount, sha512.Size, sha512.New)
	case ScramSHA3512:
		return pbkdf2.Key(b, salt, iterationCount, sha512.Size, sha3.New512)
	}
	return nil
}

func testScramAuthHash(b []byte, scramType ScramType) []byte {
	var h hash.Hash
	switch scramType {
	case ScramSHA1:
		h = sha1.New()
	case ScramSHA256:
		h = sha256.New()
	case ScramSHA512:
		h = sha512.New()
	case ScramSHA3512:
		h = sha3.New512()
	default:
		return nil
	}
	_, _ = h.Write(b)
	return h.Sum(nil)
}

func testUser() *usermodel.User {
	// password: 1234
	var usr usermodel.User
	usr.Scram = &usermodel.Scram{}
	usr.Username = "ortuman"
	usr.Scram.Sha1 = tScramSHA1Base64
	usr.Scram.Sha256 = tScramSHA256Base64
	usr.Scram.Sha512 = tScramSHA512Base64
	usr.Scram.Sha3512 = tScramSHA3512Base64
	usr.Scram.Salt = tSaltBase64
	usr.Scram.IterationCount = tIterationCount
	usr.Scram.PepperId = "v1"
	return &usr
}

func testPeppers() *pepper.Keys {
	ks, _ := pepper.NewKeys(pepper.Config{
		Keys:  map[string]string{"v1": pepperKey},
		UseID: "v1",
	})
	return ks
}

func randomBytes(l int) []byte {
	b := make([]byte, l)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}
