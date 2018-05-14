/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package server

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/memstorage"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/util"
	"github.com/ortuman/jackal/xml"
	"github.com/stretchr/testify/require"
)

type digestMD5AuthTestHelper struct {
	t        *testing.T
	testStrm c2s.Stream
	authr    *digestMD5Authenticator
}

func (h *digestMD5AuthTestHelper) clientParamsFromChallenge(challenge string) *digestMD5Parameters {
	b, err := base64.StdEncoding.DecodeString(challenge)
	require.Nil(h.t, err)
	srvParams := h.authr.parseParameters(string(b))
	clParams := *srvParams
	clParams.setParameter("cnonce=" + hex.EncodeToString(util.RandomBytes(16)))
	clParams.setParameter("username=" + h.testStrm.Username())
	clParams.setParameter("realm=" + h.testStrm.Domain())
	clParams.setParameter("nc=00000001")
	clParams.setParameter("qop=auth")
	clParams.setParameter("digest-uri=" + fmt.Sprintf("xmpp/%s", h.testStrm.Domain()))
	clParams.setParameter("charset=utf-8")
	clParams.setParameter("authzid=test")
	return &clParams
}

func (h *digestMD5AuthTestHelper) sendClientParamsResponse(params *digestMD5Parameters) error {
	response := xml.NewElementNamespace("response", "urn:ietf:params:xml:ns:xmpp-sasl")
	response.SetText(h.serializeParams(params))
	return h.authr.ProcessElement(response)
}

func (h *digestMD5AuthTestHelper) serializeParams(params *digestMD5Parameters) string {
	fmtStr := `username="%s",realm="%s",nonce="%s",cnonce="%s",nc=%s,qop=%s,digest-uri="%s",response=%s,charset=%s`
	str := fmt.Sprintf(fmtStr, params.username, params.realm, params.nonce, params.cnonce, params.nc, params.qop,
		params.digestURI, params.response, params.charset)
	if len(params.servType) > 0 {
		str += ",serv-type=" + params.servType
	}
	if len(params.authID) > 0 {
		str += ",authzid=" + params.authID
	}
	return base64.StdEncoding.EncodeToString([]byte(str))
}

func TestDigesMD5Authentication(t *testing.T) {
	user := &model.User{Username: "mariana", Password: "1234"}
	testStrm := authTestSetup(user)
	defer authTestTeardown()

	authr := newDigestMD5(testStrm)
	require.Equal(t, authr.Mechanism(), "DIGEST-MD5")
	require.False(t, authr.UsesChannelBinding())

	// test garbage input...
	require.Equal(t, authr.ProcessElement(xml.NewElementName("garbage")), errSASLNotAuthorized)

	helper := digestMD5AuthTestHelper{t: t, testStrm: testStrm, authr: authr}

	auth := xml.NewElementNamespace("auth", "urn:ietf:params:xml:ns:xmpp-sasl")
	auth.SetAttribute("mechanism", "DIGEST-MD5")
	authr.ProcessElement(auth)

	challenge := testStrm.FetchElement()
	require.Equal(t, challenge.Name(), "challenge")
	clParams := helper.clientParamsFromChallenge(challenge.Text())
	clientResp := authr.computeResponse(clParams, user, true)
	clParams.setParameter("response=" + clientResp)
	clParams.response = clientResp

	// empty payload
	response := xml.NewElementNamespace("response", "urn:ietf:params:xml:ns:xmpp-sasl")
	response.SetText("")
	require.Equal(t, errSASLMalformedRequest, authr.ProcessElement(response))

	// incorrect payload encoding
	response.SetText("bad_payload")
	require.Equal(t, errSASLIncorrectEncoding, authr.ProcessElement(response))

	// invalid username...
	cl0 := *clParams
	cl0.setParameter("username=mariana-inv")
	require.Equal(t, errSASLNotAuthorized, helper.sendClientParamsResponse(&cl0))

	// invalid realm...
	cl1 := *clParams
	cl1.setParameter("realm=localhost-inv")
	require.Equal(t, errSASLNotAuthorized, helper.sendClientParamsResponse(&cl1))

	// invalid nc...
	cl2 := *clParams
	cl2.setParameter("nc=00000001-inv")
	require.Equal(t, errSASLNotAuthorized, helper.sendClientParamsResponse(&cl2))

	// invalid nc...
	cl3 := *clParams
	cl3.setParameter("qop=auth-inv")
	require.Equal(t, errSASLNotAuthorized, helper.sendClientParamsResponse(&cl3))

	// invalid serv-type...
	cl4 := *clParams
	cl4.setParameter("serv-type=http")
	require.Equal(t, errSASLNotAuthorized, helper.sendClientParamsResponse(&cl4))

	// invalid digest-uri...
	cl5 := *clParams
	cl5.setParameter("digest-uri=http/localhost")
	require.Equal(t, errSASLNotAuthorized, helper.sendClientParamsResponse(&cl5))

	cl6 := *clParams
	cl6.setParameter("digest-uri=xmpp/localhost-inv")
	require.Equal(t, errSASLNotAuthorized, helper.sendClientParamsResponse(&cl6))

	// invalid password...
	cl7 := *clParams
	user2 := &model.User{Username: "mariana", Password: "bad_password"}
	badClientResp := authr.computeResponse(&cl7, user2, true)
	cl7.setParameter("response=" + badClientResp)
	require.Equal(t, errSASLNotAuthorized, helper.sendClientParamsResponse(&cl7))

	// storage error...
	storage.ActivateMockedError()
	require.Equal(t, memstorage.ErrMockedError, helper.sendClientParamsResponse(clParams))

	// successful authentication...
	storage.DeactivateMockedError()
	require.Nil(t, helper.sendClientParamsResponse(clParams))

	challenge = testStrm.FetchElement()

	serverResp := authr.computeResponse(clParams, user, false)
	require.Equal(t, base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("rspauth=%s", serverResp))), challenge.Text())

	response.SetText("")
	authr.ProcessElement(response)

	success := testStrm.FetchElement()
	require.Equal(t, "success", success.Name())

	// successfully authenticated
	require.True(t, authr.Authenticated())
	require.Equal(t, "mariana", authr.Username())

	// already authenticated...
	require.Nil(t, authr.ProcessElement(auth))

	// test reset
	authr.Reset()
	require.Equal(t, authr.state, startDigestMD5State)
	require.False(t, authr.Authenticated())
	require.Equal(t, "", authr.Username())
}
