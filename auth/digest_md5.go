/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package auth

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/util"
	"github.com/ortuman/jackal/xmpp"
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

// DigestMD5 represents a DIGEST-MD5 authenticator.
type DigestMD5 struct {
	stm           stream.C2S
	state         digestMD5State
	username      string
	authenticated bool
}

// NewDigestMD5 returns a new digest-md5 authenticator instance.
func NewDigestMD5(stm stream.C2S) *DigestMD5 {
	return &DigestMD5{
		stm:   stm,
		state: startDigestMD5State,
	}
}

// Mechanism returns authenticator mechanism name.
func (d *DigestMD5) Mechanism() string {
	return "DIGEST-MD5"
}

// Username returns authenticated username in case
// authentication process has been completed.
func (d *DigestMD5) Username() string {
	return d.username
}

// Authenticated returns whether or not user has been authenticated.
func (d *DigestMD5) Authenticated() bool {
	return d.authenticated
}

// UsesChannelBinding returns whether or not digest-md5 authenticator
// requires channel binding bytes.
func (d *DigestMD5) UsesChannelBinding() bool {
	return false
}

// ProcessElement process an incoming authenticator element.
func (d *DigestMD5) ProcessElement(ctx context.Context, elem xmpp.XElement) error {
	if d.Authenticated() {
		return nil
	}
	switch elem.Name() {
	case "auth":
		switch d.state {
		case startDigestMD5State:
			return d.handleStart(ctx)
		}
	case "response":
		switch d.state {
		case challengedDigestMD5State:
			return d.handleChallenged(ctx, elem)
		case authenticatedDigestMD5State:
			return d.handleAuthenticated(ctx)
		}
	}
	return ErrSASLNotAuthorized
}

// Reset resets digest-md5 authenticator internal state.
func (d *DigestMD5) Reset() {
	d.state = startDigestMD5State
	d.username = ""
	d.authenticated = false
}

func (d *DigestMD5) handleStart(ctx context.Context) error {
	domain := d.stm.Domain()
	nonce := base64.StdEncoding.EncodeToString(util.RandomBytes(32))
	chnge := fmt.Sprintf(`realm="%s",nonce="%s",qop="auth",charset=utf-8,algorithm=md5-sess`, domain, nonce)

	respElem := xmpp.NewElementNamespace("challenge", saslNamespace)
	respElem.SetText(base64.StdEncoding.EncodeToString([]byte(chnge)))
	d.stm.SendElement(ctx, respElem)

	d.state = challengedDigestMD5State
	return nil
}

func (d *DigestMD5) handleChallenged(ctx context.Context, elem xmpp.XElement) error {
	if len(elem.Text()) == 0 {
		return ErrSASLMalformedRequest
	}
	b, err := base64.StdEncoding.DecodeString(elem.Text())
	if err != nil {
		return ErrSASLIncorrectEncoding
	}
	params := d.parseParameters(string(b))

	// validate realm
	if params.realm != d.stm.Domain() {
		return ErrSASLNotAuthorized
	}
	// validate nc
	if params.nc != "00000001" {
		return ErrSASLNotAuthorized
	}
	// validate qop
	if params.qop != "auth" {
		return ErrSASLNotAuthorized
	}
	// validate serv-type
	if len(params.servType) > 0 && params.servType != "xmpp" {
		return ErrSASLNotAuthorized
	}
	// validate digest-uri
	if !strings.HasPrefix(params.digestURI, "xmpp/") || params.digestURI[5:] != d.stm.Domain() {
		return ErrSASLNotAuthorized
	}
	// validate user
	user, err := storage.FetchUser(ctx, params.username)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrSASLNotAuthorized
	}
	// validate response
	clientResp := d.computeResponse(params, user, true)
	if clientResp != params.response {
		return ErrSASLNotAuthorized
	}

	// authenticated... compute and send server response
	serverResp := d.computeResponse(params, user, false)
	respAuth := fmt.Sprintf("rspauth=%s", serverResp)

	respElem := xmpp.NewElementNamespace("challenge", saslNamespace)
	respElem.SetText(base64.StdEncoding.EncodeToString([]byte(respAuth)))
	d.stm.SendElement(ctx, respElem)

	d.username = user.Username
	d.state = authenticatedDigestMD5State
	return nil
}

func (d *DigestMD5) handleAuthenticated(ctx context.Context) error {
	d.authenticated = true
	d.stm.SendElement(ctx, xmpp.NewElementNamespace("success", saslNamespace))
	return nil
}

func (d *DigestMD5) parseParameters(str string) *digestMD5Parameters {
	params := &digestMD5Parameters{}
	s := strings.Split(str, ",")
	for i := 0; i < len(s); i++ {
		params.setParameter(s[i])
	}
	return params
}

func (d *DigestMD5) computeResponse(params *digestMD5Parameters, user *model.User, asClient bool) string {
	x := params.username + ":" + params.realm + ":" + user.Password
	y := d.md5Hash([]byte(x))

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

	ha1 := hex.EncodeToString(d.md5Hash(a1.Bytes()))
	ha2 := hex.EncodeToString(d.md5Hash(a2.Bytes()))

	kd := ha1
	kd += ":" + params.nonce
	kd += ":" + params.nc
	kd += ":" + params.cnonce
	kd += ":" + params.qop
	kd += ":" + ha2
	return hex.EncodeToString(d.md5Hash([]byte(kd)))
}

func (d *DigestMD5) md5Hash(b []byte) []byte {
	hasher := md5.New()
	hasher.Write(b)
	return hasher.Sum(nil)
}
