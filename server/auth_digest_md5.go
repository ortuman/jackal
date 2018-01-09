/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package server

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/entity"
	"github.com/ortuman/jackal/util"
	"github.com/ortuman/jackal/xml"
)

type digestMD5State int

const (
	startDigestMD5State digestMD5State = iota
	challengedDigestMD5State
	authenticatedDigestMD5State
)

type digestMD5Parameters struct {
	username  string
	realm     string
	nonce     string
	cnonce    string
	nc        string
	qop       string
	servType  string
	host      string
	digestURI string
	response  string
	charset   string
	authID    string
}

func (r *digestMD5Parameters) setParameter(p string) {
	key, val := util.SplitKeyAndValue(p, '=')

	// strip value double quotes
	val = strings.TrimPrefix(val, `"`)
	val = strings.TrimSuffix(val, `"`)

	switch key {
	case "username":
		r.username = val
	case "realm":
		r.realm = val
	case "nonce":
		r.nonce = val
	case "cnonce":
		r.cnonce = val
	case "nc":
		r.nc = val
	case "qop":
		r.qop = val
	case "serv-type":
		r.servType = val
	case "host":
		r.host = val
	case "digest-uri":
		r.digestURI = val
	case "response":
		r.response = val
	case "charset":
		r.charset = val
	case "authzid":
		r.authID = val
	}
}

type digestMD5Authenticator struct {
	strm          *serverStream
	state         digestMD5State
	username      string
	authenticated bool
}

func newDigestMD5(strm *serverStream) authenticator {
	return &digestMD5Authenticator{
		strm:  strm,
		state: startDigestMD5State,
	}
}

func (d *digestMD5Authenticator) Mechanism() string {
	return "DIGEST-MD5"
}

func (d *digestMD5Authenticator) Username() string {
	return d.username
}

func (d *digestMD5Authenticator) Authenticated() bool {
	return d.authenticated
}

func (d *digestMD5Authenticator) UsesChannelBinding() bool {
	return false
}

func (d *digestMD5Authenticator) ProcessElement(elem xml.Element) error {
	if d.Authenticated() {
		return nil
	}
	switch elem.Name() {
	case "auth":
		switch d.state {
		case startDigestMD5State:
			return d.handleStart(elem)
		}
	case "response":
		switch d.state {
		case challengedDigestMD5State:
			return d.handleChallenged(elem)
		case authenticatedDigestMD5State:
			return d.handleAuthenticated(elem)
		}
	}
	return errSASLNotAuthorized
}

func (d *digestMD5Authenticator) Reset() {
	d.state = startDigestMD5State
	d.username = ""
	d.authenticated = false
}

func (d *digestMD5Authenticator) handleStart(elem xml.Element) error {
	if elem.TextLen() != 0 {
		return errSASLMalformedRequest
	}
	domain := d.strm.Domain()
	nonce := base64.StdEncoding.EncodeToString(util.RandomBytes(32))
	chnge := fmt.Sprintf(`realm="%s",nonce="%s",qop="auth",charset=utf-8,algorithm=md5-sess`, domain, nonce)

	respElem := xml.NewElementNamespace("challenge", saslNamespace)
	respElem.SetText(base64.StdEncoding.EncodeToString([]byte(chnge)))
	d.strm.SendElement(respElem)

	d.state = challengedDigestMD5State
	return nil
}

func (d *digestMD5Authenticator) handleChallenged(elem xml.Element) error {
	if elem.TextLen() == 0 {
		return errSASLMalformedRequest
	}
	b, err := base64.StdEncoding.DecodeString(elem.Text())
	if err != nil {
		return errSASLIncorrectEncoding
	}
	params := d.parseParameters(string(b))

	// validate realm
	if params.realm != d.strm.Domain() {
		return errSASLNotAuthorized
	}
	// validate nc
	if params.nc != "00000001" {
		return errSASLNotAuthorized
	}
	// validate qop
	if params.qop != "auth" {
		return errSASLNotAuthorized
	}
	// validate serv-type
	if len(params.servType) > 0 && params.servType != "xmpp" {
		return errSASLNotAuthorized
	}
	// validate digest-uri
	if !strings.HasPrefix(params.digestURI, "xmpp/") || params.digestURI[5:] != d.strm.Domain() {
		return errSASLNotAuthorized
	}
	// validate user
	user, err := storage.Instance().FetchUser(params.username)
	if err != nil {
		return err
	}
	if user == nil {
		return errSASLNotAuthorized
	}
	// validate response
	clientResp := computeResponse(params, user, true)
	if clientResp != params.response {
		return errSASLNotAuthorized
	}

	// authenticated... compute and send server response
	serverResp := computeResponse(params, user, false)
	respAuth := fmt.Sprintf("rspauth=%s", serverResp)

	respElem := xml.NewElementNamespace("challenge", saslNamespace)
	respElem.SetText(base64.StdEncoding.EncodeToString([]byte(respAuth)))
	d.strm.SendElement(respElem)

	d.username = user.Username
	d.state = authenticatedDigestMD5State
	return nil
}

func (d *digestMD5Authenticator) handleAuthenticated(elem xml.Element) error {
	d.authenticated = true
	d.strm.SendElement(xml.NewElementNamespace("success", saslNamespace))
	return nil
}

func (d *digestMD5Authenticator) parseParameters(str string) *digestMD5Parameters {
	params := &digestMD5Parameters{}
	s := strings.Split(str, ",")
	for i := 0; i < len(s); i++ {
		params.setParameter(s[i])
	}
	return params
}

func computeResponse(params *digestMD5Parameters, user *entity.User, asClient bool) string {
	x := params.username + ":" + params.realm + ":" + user.Password
	y := md5Hash([]byte(x))

	a1 := bytes.NewBuffer(y)
	a1.WriteString(":" + params.nonce + ":" + params.cnonce)
	if len(params.authID) > 0 {
		a1.WriteString(":" + params.authID)
	}

	var c string
	if asClient {
		c = "AUTHENTICATE"
	} else {
		c = ""
	}
	a2 := bytes.NewBuffer([]byte(c))
	a2.WriteString(":" + params.digestURI)

	ha1 := hex.EncodeToString(md5Hash(a1.Bytes()))
	ha2 := hex.EncodeToString(md5Hash(a2.Bytes()))

	kd := ha1
	kd += ":" + params.nonce
	kd += ":" + params.nc
	kd += ":" + params.cnonce
	kd += ":" + params.qop
	kd += ":" + ha2
	return hex.EncodeToString(md5Hash([]byte(kd)))
}

func md5Hash(b []byte) []byte {
	hasher := md5.New()
	hasher.Write(b)
	return hasher.Sum(nil)
}
