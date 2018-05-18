/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/module/offline"
	"github.com/ortuman/jackal/module/roster"
	"github.com/ortuman/jackal/module/xep0012"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/module/xep0049"
	"github.com/ortuman/jackal/module/xep0054"
	"github.com/ortuman/jackal/module/xep0077"
	"github.com/ortuman/jackal/module/xep0092"
	"github.com/ortuman/jackal/module/xep0191"
	"github.com/ortuman/jackal/module/xep0199"
	"github.com/ortuman/jackal/server/compress"
	"github.com/ortuman/jackal/server/transport"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/stream/errors"
	"github.com/ortuman/jackal/util"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
)

const streamMailboxSize = 64

const (
	connecting uint32 = iota
	connected
	authenticating
	authenticated
	sessionStarted
	disconnected
)

const (
	jabberClientNamespace     = "jabber:client"
	framedStreamNamespace     = "urn:ietf:params:xml:ns:xmpp-framing"
	streamNamespace           = "http://etherx.jabber.org/streams"
	tlsNamespace              = "urn:ietf:params:xml:ns:xmpp-tls"
	compressProtocolNamespace = "http://jabber.org/protocol/compress"
	bindNamespace             = "urn:ietf:params:xml:ns:xmpp-bind"
	sessionNamespace          = "urn:ietf:params:xml:ns:xmpp-session"
	blockedErrorNamespace     = "urn:xmpp:blocking:errors"
)

// stream context keys
const (
	usernameContextKey      = "username"
	domainContextKey        = "domain"
	resourceContextKey      = "resource"
	jidContextKey           = "jid"
	securedContextKey       = "secured"
	authenticatedContextKey = "authenticated"
	compressedContextKey    = "compressed"
	presenceContextKey      = "presence"
)

// once dispatch handlers
const (
	rosterOnce  = "rosterOnce"
	offlineOnce = "offlineOnce"
)

type c2sStream struct {
	cfg         *Config
	tr          transport.Transport
	parser      *xml.Parser
	id          string
	connected   uint32
	state       uint32
	ctx         *stream.Context
	authrs      []authenticator
	activeAuthr authenticator
	iqHandlers  []module.IQHandler
	roster      *roster.Roster
	discoInfo   *xep0030.DiscoInfo
	register    *xep0077.Register
	ping        *xep0199.Ping
	blockCmd    *xep0191.BlockingCommand
	offline     *offline.Offline
	actorCh     chan func()
}

func newC2SStream(id string, tr transport.Transport, cfg *Config) *c2sStream {
	s := &c2sStream{
		cfg:     cfg,
		id:      id,
		tr:      tr,
		parser:  xml.NewParser(tr, cfg.Transport.MaxStanzaSize),
		state:   connecting,
		ctx:     stream.NewContext(),
		actorCh: make(chan func(), streamMailboxSize),
	}
	// initialize stream context
	secured := !(cfg.Transport.Type == transport.Socket)
	s.ctx.SetBool(secured, securedContextKey)

	domain := c2s.Instance().DefaultLocalDomain()
	s.ctx.SetString(domain, domainContextKey)

	j, _ := xml.NewJID("", domain, "", true)
	s.ctx.SetObject(j, jidContextKey)

	// initialize authenticators
	s.initializeAuthenticators()

	// initialize register module
	if _, ok := s.cfg.Modules["registration"]; ok {
		s.register = xep0077.New(&s.cfg.ModRegistration, s, s.discoInfo)
	}

	if cfg.Transport.ConnectTimeout > 0 {
		go s.startConnectTimeoutTimer(cfg.Transport.ConnectTimeout)
	}
	go s.actorLoop()
	go s.doRead() // start reading transport...

	return s
}

// ID returns stream identifier.
func (s *c2sStream) ID() string {
	return s.id
}

// Context returns stream associated context.
func (s *c2sStream) Context() *stream.Context {
	return s.ctx
}

// Username returns current stream username.
func (s *c2sStream) Username() string {
	return s.ctx.String(usernameContextKey)
}

// Domain returns current stream domain.
func (s *c2sStream) Domain() string {
	return s.ctx.String(domainContextKey)
}

// Resource returns current stream resource.
func (s *c2sStream) Resource() string {
	return s.ctx.String(resourceContextKey)
}

// JID returns current user JID.
func (s *c2sStream) JID() *xml.JID {
	return s.ctx.Object(jidContextKey).(*xml.JID)
}

// IsAuthenticated returns whether or not the XMPP stream
// has successfully authenticated.
func (s *c2sStream) IsAuthenticated() bool {
	return s.ctx.Bool(authenticatedContextKey)
}

// IsSecured returns whether or not the XMPP stream
// has been secured using SSL/TLS.
func (s *c2sStream) IsSecured() bool {
	return s.ctx.Bool(securedContextKey)
}

// IsCompressed returns whether or not the XMPP stream
// has enabled a compression method.
func (s *c2sStream) IsCompressed() bool {
	return s.ctx.Bool(compressedContextKey)
}

// Presence returns last sent presence element.
func (s *c2sStream) Presence() *xml.Presence {
	switch v := s.ctx.Object(presenceContextKey).(type) {
	case *xml.Presence:
		return v
	}
	return nil
}

// SendElement sends the given XML element.
func (s *c2sStream) SendElement(element xml.XElement) {
	s.actorCh <- func() {
		s.writeElement(element)
	}
}

// Disconnect disconnects remote peer by closing
// the underlying TCP socket connection.
func (s *c2sStream) Disconnect(err error) {
	s.actorCh <- func() {
		s.disconnect(err)
	}
}

func (s *c2sStream) initializeAuthenticators() {
	for _, a := range s.cfg.SASL {
		switch a {
		case "plain":
			s.authrs = append(s.authrs, newPlainAuthenticator(s))

		case "digest_md5":
			s.authrs = append(s.authrs, newDigestMD5(s))

		case "scram_sha_1":
			s.authrs = append(s.authrs, newScram(s, s.tr, sha1ScramType, false))
			s.authrs = append(s.authrs, newScram(s, s.tr, sha1ScramType, true))

		case "scram_sha_256":
			s.authrs = append(s.authrs, newScram(s, s.tr, sha256ScramType, false))
			s.authrs = append(s.authrs, newScram(s, s.tr, sha256ScramType, true))
		}
	}
}

func (s *c2sStream) initializeModules() {
	// XEP-0030: Service Discovery (https://xmpp.org/extensions/xep-0030.html)
	s.discoInfo = xep0030.New(s)
	s.iqHandlers = append(s.iqHandlers, s.discoInfo)

	// register default disco info entities
	s.discoInfo.RegisterEntity(s.Domain(), "")
	s.discoInfo.RegisterEntity(s.JID().ToBareJID().String(), "")

	// Roster (https://xmpp.org/rfcs/rfc3921.html#roster)
	s.roster = roster.New(&s.cfg.ModRoster, s)
	s.iqHandlers = append(s.iqHandlers, s.roster)

	// XEP-0012: Last Activity (https://xmpp.org/extensions/xep-0012.html)
	if _, ok := s.cfg.Modules["last_activity"]; ok {
		s.iqHandlers = append(s.iqHandlers, xep0012.New(s, s.discoInfo))
	}

	// XEP-0049: Private XML Storage (https://xmpp.org/extensions/xep-0049.html)
	if _, ok := s.cfg.Modules["private"]; ok {
		s.iqHandlers = append(s.iqHandlers, xep0049.New(s))
	}

	// XEP-0054: vcard-temp (https://xmpp.org/extensions/xep-0054.html)
	if _, ok := s.cfg.Modules["vcard"]; ok {
		s.iqHandlers = append(s.iqHandlers, xep0054.New(s, s.discoInfo))
	}

	// XEP-0077: In-band registration (https://xmpp.org/extensions/xep-0077.html)
	if s.register != nil {
		s.iqHandlers = append(s.iqHandlers, s.register)
	}

	// XEP-0092: Software Version (https://xmpp.org/extensions/xep-0092.html)
	if _, ok := s.cfg.Modules["version"]; ok {
		s.iqHandlers = append(s.iqHandlers, xep0092.New(&s.cfg.ModVersion, s, s.discoInfo))
	}

	// XEP-0191: Blocking Command (https://xmpp.org/extensions/xep-0191.html)
	if _, ok := s.cfg.Modules["blocking_command"]; ok {
		s.blockCmd = xep0191.New(s, s.discoInfo)
		s.iqHandlers = append(s.iqHandlers, s.blockCmd)
	}

	// XEP-0199: XMPP Ping (https://xmpp.org/extensions/xep-0199.html)
	if _, ok := s.cfg.Modules["ping"]; ok {
		s.ping = xep0199.New(&s.cfg.ModPing, s, s.discoInfo)
		s.iqHandlers = append(s.iqHandlers, s.ping)
	}

	// XEP-0160: Offline message storage (https://xmpp.org/extensions/xep-0160.html)
	if _, ok := s.cfg.Modules["offline"]; ok {
		s.offline = offline.New(&s.cfg.ModOffline, s, s.discoInfo)
	}
}

func (s *c2sStream) startConnectTimeoutTimer(timeoutInSeconds int) {
	tr := time.NewTimer(time.Second * time.Duration(timeoutInSeconds))
	<-tr.C
	if atomic.LoadUint32(&s.connected) == 0 {
		// connection timeout...
		s.actorCh <- func() {
			s.disconnect(streamerror.ErrConnectionTimeout)
		}
	}
}

func (s *c2sStream) handleElement(elem xml.XElement) {
	isWebSocketTr := s.cfg.Transport.Type == transport.WebSocket
	if isWebSocketTr && elem.Name() == "close" && elem.Namespace() == framedStreamNamespace {
		s.disconnect(nil)
		return
	}
	switch s.getState() {
	case connecting:
		s.handleConnecting(elem)
	case connected:
		s.handleConnected(elem)
	case authenticated:
		s.handleAuthenticated(elem)
	case authenticating:
		s.handleAuthenticating(elem)
	case sessionStarted:
		s.handleSessionStarted(elem)
	default:
		break
	}
}

func (s *c2sStream) handleConnecting(elem xml.XElement) {
	// activate 'connected' flag
	atomic.StoreUint32(&s.connected, 1)

	// validate stream element
	if err := s.validateStreamElement(elem); err != nil {
		s.disconnectWithStreamError(err)
		return
	}
	// assign stream domain
	s.ctx.SetString(elem.To(), domainContextKey)

	// open stream
	s.openStream()

	features := xml.NewElementName("stream:features")
	features.SetAttribute("xmlns:stream", streamNamespace)
	features.SetAttribute("version", "1.0")

	if !s.IsAuthenticated() {
		features.AppendElements(s.unauthenticatedFeatures())
		s.setState(connected)
	} else {
		features.AppendElements(s.authenticatedFeatures())
		s.setState(authenticated)
	}
	s.writeElement(features)
}

func (s *c2sStream) unauthenticatedFeatures() []xml.XElement {
	var features []xml.XElement

	isSocketTransport := s.cfg.Transport.Type == transport.Socket

	if isSocketTransport && !s.IsSecured() {
		startTLS := xml.NewElementName("starttls")
		startTLS.SetNamespace("urn:ietf:params:xml:ns:xmpp-tls")
		startTLS.AppendElement(xml.NewElementName("required"))
		features = append(features, startTLS)
	}

	// attach SASL mechanisms
	shouldOfferSASL := (!isSocketTransport || (isSocketTransport && s.IsSecured()))

	if shouldOfferSASL && len(s.authrs) > 0 {
		mechanisms := xml.NewElementName("mechanisms")
		mechanisms.SetNamespace(saslNamespace)
		for _, athr := range s.authrs {
			mechanism := xml.NewElementName("mechanism")
			mechanism.SetText(athr.Mechanism())
			mechanisms.AppendElement(mechanism)
		}
		features = append(features, mechanisms)
	}

	// allow In-band registration over encrypted stream only
	allowRegistration := s.IsSecured()

	if _, ok := s.cfg.Modules["registration"]; ok && allowRegistration {
		registerFeature := xml.NewElementNamespace("register", "http://jabber.org/features/iq-register")
		features = append(features, registerFeature)
	}
	return features
}

func (s *c2sStream) authenticatedFeatures() []xml.XElement {
	var features []xml.XElement

	isSocketTransport := s.cfg.Transport.Type == transport.Socket

	// attach compression feature
	compressionAvailable := isSocketTransport && s.cfg.Compression.Level != compress.NoCompression

	if !s.IsCompressed() && compressionAvailable {
		compression := xml.NewElementNamespace("compression", "http://jabber.org/features/compress")
		method := xml.NewElementName("method")
		method.SetText("zlib")
		compression.AppendElement(method)
		features = append(features, compression)
	}
	bind := xml.NewElementNamespace("bind", "urn:ietf:params:xml:ns:xmpp-bind")
	bind.AppendElement(xml.NewElementName("required"))
	features = append(features, bind)

	session := xml.NewElementNamespace("session", "urn:ietf:params:xml:ns:xmpp-session")
	features = append(features, session)

	if s.roster != nil && s.cfg.ModRoster.Versioning {
		ver := xml.NewElementNamespace("ver", "urn:xmpp:features:rosterver")
		features = append(features, ver)
	}
	return features
}

func (s *c2sStream) handleConnected(elem xml.XElement) {
	switch elem.Name() {
	case "starttls":
		if len(elem.Namespace()) > 0 && elem.Namespace() != tlsNamespace {
			s.disconnectWithStreamError(streamerror.ErrInvalidNamespace)
			return
		}
		s.proceedStartTLS()

	case "auth":
		if elem.Namespace() != saslNamespace {
			s.disconnectWithStreamError(streamerror.ErrInvalidNamespace)
			return
		}
		s.startAuthentication(elem)

	case "iq":
		stanza, err := s.buildStanza(elem, false)
		if err != nil {
			s.handleElementError(elem, err)
			return
		}
		iq := stanza.(*xml.IQ)

		if s.register != nil && s.register.MatchesIQ(iq) {
			s.register.ProcessIQ(iq)
			return

		} else if iq.Elements().ChildNamespace("query", "jabber:iq:auth") != nil {
			// don't allow non-SASL authentication
			s.writeElement(iq.ServiceUnavailableError())
			return
		}
		fallthrough

	case "message", "presence":
		s.disconnectWithStreamError(streamerror.ErrNotAuthorized)

	default:
		s.disconnectWithStreamError(streamerror.ErrUnsupportedStanzaType)
	}
}

func (s *c2sStream) handleAuthenticating(elem xml.XElement) {
	if elem.Namespace() != saslNamespace {
		s.disconnectWithStreamError(streamerror.ErrInvalidNamespace)
		return
	}
	authr := s.activeAuthr
	s.continueAuthentication(elem, authr)
	if authr.Authenticated() {
		s.finishAuthentication(authr.Username())
	}
}

func (s *c2sStream) handleAuthenticated(elem xml.XElement) {
	switch elem.Name() {
	case "compress":
		if elem.Namespace() != compressProtocolNamespace {
			s.disconnectWithStreamError(streamerror.ErrUnsupportedStanzaType)
			return
		}
		s.compress(elem)

	case "iq":
		stanza, err := s.buildStanza(elem, true)
		if err != nil {
			s.handleElementError(elem, err)
			return
		}
		iq := stanza.(*xml.IQ)

		if len(s.Resource()) == 0 { // expecting bind
			s.bindResource(iq)
		} else { // expecting session
			s.startSession(iq)
		}

	default:
		s.disconnectWithStreamError(streamerror.ErrUnsupportedStanzaType)
	}
}

func (s *c2sStream) handleSessionStarted(elem xml.XElement) {
	// reset ping timer deadline
	if s.ping != nil {
		s.ping.ResetDeadline()
	}

	stanza, err := s.buildStanza(elem, true)
	if err != nil {
		s.handleElementError(elem, err)
		return
	}
	if s.isComponentDomain(stanza.ToJID().Domain()) {
		s.processComponentStanza(stanza)
	} else {
		s.processStanza(stanza)
	}
}

func (s *c2sStream) proceedStartTLS() {
	if s.IsSecured() {
		s.disconnectWithStreamError(streamerror.ErrNotAuthorized)
		return
	}
	tlsCfg, err := util.LoadCertificate(s.cfg.TLS.PrivKeyFile, s.cfg.TLS.CertFile, s.Domain())
	if err != nil {
		log.Error(err)
		s.writeElement(xml.NewElementNamespace("failure", tlsNamespace))
		s.disconnectClosingStream(true)
		return
	}
	s.ctx.SetBool(true, securedContextKey)

	s.writeElement(xml.NewElementNamespace("proceed", tlsNamespace))

	s.tr.StartTLS(tlsCfg)

	log.Infof("secured stream... id: %s", s.id)

	s.restart()
}

func (s *c2sStream) compress(elem xml.XElement) {
	if s.IsCompressed() {
		s.disconnectWithStreamError(streamerror.ErrUnsupportedStanzaType)
		return
	}
	method := elem.Elements().Child("method")
	if method == nil || len(method.Text()) == 0 {
		failure := xml.NewElementNamespace("failure", compressProtocolNamespace)
		failure.AppendElement(xml.NewElementName("setup-failed"))
		s.writeElement(failure)
		return
	}
	if method.Text() != "zlib" {
		failure := xml.NewElementNamespace("failure", compressProtocolNamespace)
		failure.AppendElement(xml.NewElementName("unsupported-method"))
		s.writeElement(failure)
		return
	}
	s.ctx.SetBool(true, compressedContextKey)

	s.writeElement(xml.NewElementNamespace("compressed", compressProtocolNamespace))

	s.tr.EnableCompression(s.cfg.Compression.Level)

	log.Infof("compressed stream... id: %s", s.id)

	s.restart()
}

func (s *c2sStream) startAuthentication(elem xml.XElement) {
	mechanism := elem.Attributes().Get("mechanism")
	for _, authr := range s.authrs {
		if authr.Mechanism() == mechanism {
			if err := s.continueAuthentication(elem, authr); err != nil {
				return
			}
			if authr.Authenticated() {
				s.finishAuthentication(authr.Username())
			} else {
				s.activeAuthr = authr
				s.setState(authenticating)
			}
			return
		}
	}

	// ...mechanism not found...
	failure := xml.NewElementNamespace("failure", saslNamespace)
	failure.AppendElement(xml.NewElementName("invalid-mechanism"))
	s.writeElement(failure)
}

func (s *c2sStream) continueAuthentication(elem xml.XElement, authr authenticator) error {
	err := authr.ProcessElement(elem)
	if saslErr, ok := err.(saslError); ok {
		s.failAuthentication(saslErr.Element())
	} else if err != nil {
		log.Error(err)
		s.failAuthentication(errSASLTemporaryAuthFailure.(saslError).Element())
	}
	return err
}

func (s *c2sStream) finishAuthentication(username string) {
	if s.activeAuthr != nil {
		s.activeAuthr.Reset()
		s.activeAuthr = nil
	}
	j, _ := xml.NewJID(username, s.Domain(), "", true)

	s.ctx.SetString(username, usernameContextKey)
	s.ctx.SetBool(true, authenticatedContextKey)
	s.ctx.SetObject(j, jidContextKey)

	s.restart()
}

func (s *c2sStream) failAuthentication(elem xml.XElement) {
	failure := xml.NewElementNamespace("failure", saslNamespace)
	failure.AppendElement(elem)
	s.writeElement(failure)

	if s.activeAuthr != nil {
		s.activeAuthr.Reset()
		s.activeAuthr = nil
	}
	s.setState(connected)
}

func (s *c2sStream) bindResource(iq *xml.IQ) {
	bind := iq.Elements().ChildNamespace("bind", bindNamespace)
	if bind == nil {
		s.writeElement(iq.NotAllowedError())
		return
	}
	var resource string
	if resourceElem := bind.Elements().Child("resource"); resourceElem != nil {
		resource = resourceElem.Text()
	} else {
		resource = uuid.New()
	}
	// try binding...
	var stm c2s.Stream
	stms := c2s.Instance().StreamsMatchingJID(s.JID().ToBareJID())
	for _, s := range stms {
		if s.Resource() == resource {
			stm = s
		}
	}

	if stm != nil {
		switch s.cfg.ResourceConflict {
		case Override:
			// override the resource with a server-generated resourcepart...
			h := sha256.New()
			h.Write([]byte(s.ID()))
			resource = hex.EncodeToString(h.Sum(nil))
		case Replace:
			// terminate the session of the currently connected client...
			stm.Disconnect(streamerror.ErrResourceConstraint)
		default:
			// disallow resource binding attempt...
			s.writeElement(iq.ConflictError())
			return
		}
	}
	userJID, err := xml.NewJID(s.Username(), s.Domain(), resource, false)
	if err != nil {
		s.writeElement(iq.BadRequestError())
		return
	}
	s.ctx.SetString(resource, resourceContextKey)
	s.ctx.SetObject(userJID, jidContextKey)

	log.Infof("binded resource... (%s/%s)", s.Username(), s.Resource())

	//...notify successful binding
	result := xml.NewIQType(iq.ID(), xml.ResultType)
	result.SetNamespace(iq.Namespace())

	binded := xml.NewElementNamespace("bind", bindNamespace)
	jid := xml.NewElementName("jid")
	jid.SetText(s.Username() + "@" + s.Domain() + "/" + s.Resource())
	binded.AppendElement(jid)
	result.AppendElement(binded)

	s.writeElement(result)

	if err := c2s.Instance().AuthenticateStream(s); err != nil {
		log.Error(err)
	}
}

func (s *c2sStream) startSession(iq *xml.IQ) {
	if len(s.Resource()) == 0 {
		// not binded yet...
		s.Disconnect(streamerror.ErrNotAuthorized)
		return
	}
	sess := iq.Elements().ChildNamespace("session", sessionNamespace)
	if sess == nil {
		s.writeElement(iq.NotAllowedError())
		return
	}
	s.writeElement(iq.ResultIQ())

	// initialize modules
	s.initializeModules()

	if s.ping != nil {
		s.ping.StartPinging()
	}
	s.setState(sessionStarted)
}

func (s *c2sStream) processStanza(stanza xml.Stanza) {
	toJID := stanza.ToJID()
	if s.isBlockedJID(toJID) { // blocked JID?
		blocked := xml.NewElementNamespace("blocked", blockedErrorNamespace)
		resp := xml.NewErrorElementFromElement(stanza, xml.ErrNotAcceptable.(*xml.StanzaError), []xml.XElement{blocked})
		s.writeElement(resp)
		return
	}
	switch stanza := stanza.(type) {
	case *xml.Presence:
		s.processPresence(stanza)
	case *xml.IQ:
		s.processIQ(stanza)
	case *xml.Message:
		s.processMessage(stanza)
	}
}

func (s *c2sStream) processComponentStanza(stanza xml.Stanza) {
}

func (s *c2sStream) processIQ(iq *xml.IQ) {
	toJID := iq.ToJID()
	if !c2s.Instance().IsLocalDomain(toJID.Domain()) {
		// TODO(ortuman): Implement XMPP federation
		return
	}
	if node := toJID.Node(); len(node) > 0 && c2s.Instance().IsBlockedJID(s.JID(), node) {
		// destination user blocked stream JID
		if iq.IsGet() || iq.IsSet() {
			s.writeElement(iq.ServiceUnavailableError())
		}
		return
	}
	if toJID.IsFullWithUser() {
		switch c2s.Instance().Route(iq) {
		case c2s.ErrResourceNotFound:
			s.writeElement(iq.ServiceUnavailableError())
		}
		return
	}

	for _, handler := range s.iqHandlers {
		if !handler.MatchesIQ(iq) {
			continue
		}
		handler.ProcessIQ(iq)
		return
	}

	// ...IQ not handled...
	if iq.IsGet() || iq.IsSet() {
		s.writeElement(iq.ServiceUnavailableError())
	}
}

func (s *c2sStream) processPresence(presence *xml.Presence) {
	toJID := presence.ToJID()
	if !c2s.Instance().IsLocalDomain(toJID.Domain()) {
		// TODO(ortuman): Implement XMPP federation
		return
	}
	if toJID.IsBare() && (toJID.Node() != s.Username() || toJID.Domain() != s.Domain()) {
		if s.roster != nil {
			s.roster.ProcessPresence(presence)
		}
		return
	}
	if toJID.IsFullWithUser() {
		c2s.Instance().Route(presence)
		return
	}
	// set context presence
	s.ctx.SetObject(presence, presenceContextKey)

	// deliver pending approval notifications
	if s.roster != nil {
		s.ctx.DoOnce(rosterOnce, func() {
			s.roster.DeliverPendingApprovalNotifications()
			s.roster.ReceivePresences()
		})
		s.roster.BroadcastPresence(presence)
	}

	// deliver offline messages
	if p := s.Presence(); s.offline != nil && p != nil && p.Priority() >= 0 {
		s.ctx.DoOnce(offlineOnce, func() {
			s.offline.DeliverOfflineMessages()
		})
	}
}

func (s *c2sStream) processMessage(message *xml.Message) {
	toJID := message.ToJID()
	if !c2s.Instance().IsLocalDomain(toJID.Domain()) {
		// TODO(ortuman): Implement XMPP federation
		return
	}

sendMessage:
	err := c2s.Instance().Route(message)
	switch err {
	case nil:
		break
	case c2s.ErrNotAuthenticated:
		if s.offline != nil {
			if (message.IsChat() || message.IsGroupChat()) && message.IsMessageWithBody() {
				return
			}
			s.offline.ArchiveMessage(message)
		}
	case c2s.ErrResourceNotFound:
		// treat the stanza as if it were addressed to <node@domain>
		toJID = toJID.ToBareJID()
		goto sendMessage
	case c2s.ErrNotExistingAccount, c2s.ErrBlockedJID:
		s.writeElement(message.ServiceUnavailableError())
	default:
		log.Error(err)
	}
}

func (s *c2sStream) actorLoop() {
	for {
		f := <-s.actorCh
		f()
		if s.getState() == disconnected {
			return
		}
	}
}

func (s *c2sStream) doRead() {
	if elem, err := s.parser.ParseElement(); err == nil {
		s.actorCh <- func() {
			s.readElement(elem)
		}
	} else {
		if s.getState() == disconnected {
			return // already disconnected...
		}

		var discErr error
		switch err {
		case nil, io.EOF, io.ErrUnexpectedEOF:
			break

		case xml.ErrStreamClosedByPeer: // ...received </stream:stream>
			if s.cfg.Transport.Type != transport.Socket {
				discErr = streamerror.ErrInvalidXML
			}

		case xml.ErrTooLargeStanza:
			discErr = streamerror.ErrPolicyViolation

		default:
			switch e := err.(type) {
			case net.Error:
				if e.Timeout() {
					discErr = streamerror.ErrConnectionTimeout
				} else {
					discErr = streamerror.ErrInvalidXML
				}

			case *websocket.CloseError:
				break // connection closed by peer...

			default:
				log.Error(err)
				discErr = streamerror.ErrInvalidXML
			}
		}
		s.actorCh <- func() {
			s.disconnect(discErr)
		}
	}
}

func (s *c2sStream) writeElement(element xml.XElement) {
	log.Debugf("SEND: %v", element)
	s.tr.WriteElement(element, true)
}

func (s *c2sStream) readElement(elem xml.XElement) {
	if elem != nil {
		log.Debugf("RECV: %v", elem)
		s.handleElement(elem)
	}
	if s.getState() != disconnected {
		go s.doRead()
	}
}

func (s *c2sStream) disconnect(err error) {
	switch err {
	case nil:
		s.disconnectClosingStream(false)
	default:
		if strmErr, ok := err.(*streamerror.Error); ok {
			s.disconnectWithStreamError(strmErr)
		} else {
			log.Error(err)
			s.disconnectClosingStream(false)
		}
	}
}

func (s *c2sStream) openStream() {
	var ops *xml.Element
	var includeClosing bool

	buf := &bytes.Buffer{}
	switch s.cfg.Transport.Type {
	case transport.Socket:
		ops = xml.NewElementName("stream:stream")
		ops.SetAttribute("xmlns", jabberClientNamespace)
		ops.SetAttribute("xmlns:stream", streamNamespace)
		buf.WriteString(`<?xml version="1.0"?>`)

	case transport.WebSocket:
		ops = xml.NewElementName("open")
		ops.SetAttribute("xmlns", framedStreamNamespace)
		includeClosing = true

	default:
		return
	}
	ops.SetAttribute("id", uuid.New())
	ops.SetAttribute("from", s.Domain())
	ops.SetAttribute("version", "1.0")
	ops.ToXML(buf, includeClosing)

	openStr := buf.String()
	log.Debugf("SEND: %s", openStr)

	s.tr.WriteString(buf.String())
}

func (s *c2sStream) buildStanza(elem xml.XElement, validateFrom bool) (xml.Stanza, error) {
	if err := s.validateNamespace(elem); err != nil {
		return nil, err
	}
	fromJID, toJID, err := s.extractAddresses(elem, validateFrom)
	if err != nil {
		return nil, err
	}
	switch elem.Name() {
	case "iq":
		iq, err := xml.NewIQFromElement(elem, fromJID, toJID)
		if err != nil {
			log.Error(err)
			return nil, xml.ErrBadRequest
		}
		return iq, nil

	case "presence":
		presence, err := xml.NewPresenceFromElement(elem, fromJID, toJID)
		if err != nil {
			log.Error(err)
			return nil, xml.ErrBadRequest
		}
		return presence, nil

	case "message":
		message, err := xml.NewMessageFromElement(elem, fromJID, toJID)
		if err != nil {
			log.Error(err)
			return nil, xml.ErrBadRequest
		}
		return message, nil
	}
	return nil, streamerror.ErrUnsupportedStanzaType
}

func (s *c2sStream) handleElementError(elem xml.XElement, err error) {
	if streamErr, ok := err.(*streamerror.Error); ok {
		s.disconnectWithStreamError(streamErr)
	} else if stanzaErr, ok := err.(*xml.StanzaError); ok {
		s.writeElement(xml.NewErrorElementFromElement(elem, stanzaErr, nil))
	} else {
		log.Error(err)
	}
}

func (s *c2sStream) validateStreamElement(elem xml.XElement) *streamerror.Error {
	switch s.cfg.Transport.Type {
	case transport.Socket:
		if elem.Name() != "stream:stream" {
			return streamerror.ErrUnsupportedStanzaType
		}
		if elem.Namespace() != jabberClientNamespace || elem.Attributes().Get("xmlns:stream") != streamNamespace {
			return streamerror.ErrInvalidNamespace
		}

	case transport.WebSocket:
		if elem.Name() != "open" {
			return streamerror.ErrUnsupportedStanzaType
		}
		if elem.Namespace() != framedStreamNamespace {
			return streamerror.ErrInvalidNamespace
		}
	}
	to := elem.To()
	if len(to) > 0 && !c2s.Instance().IsLocalDomain(to) {
		return streamerror.ErrHostUnknown
	}
	if elem.Version() != "1.0" {
		return streamerror.ErrUnsupportedVersion
	}
	return nil
}

func (s *c2sStream) validateNamespace(elem xml.XElement) *streamerror.Error {
	ns := elem.Namespace()
	if len(ns) == 0 || ns == jabberClientNamespace {
		return nil
	}
	return streamerror.ErrInvalidNamespace
}

func (s *c2sStream) extractAddresses(elem xml.XElement, validateFrom bool) (fromJID *xml.JID, toJID *xml.JID, err error) {
	// validate from JID
	from := elem.From()
	if validateFrom && len(from) > 0 && !s.isValidFrom(from) {
		return nil, nil, streamerror.ErrInvalidFrom
	}
	fromJID = s.JID()

	// validate to JID
	to := elem.To()
	if len(to) > 0 {
		toJID, err = xml.NewJIDString(elem.To(), false)
		if err != nil {
			return nil, nil, xml.ErrJidMalformed
		}
	} else {
		toJID = s.JID().ToBareJID() // account's bare JID as default 'to'
	}
	return
}

func (s *c2sStream) isValidFrom(from string) bool {
	validFrom := false
	j, err := xml.NewJIDString(from, false)
	if err == nil && j != nil {
		node := j.Node()
		domain := j.Domain()
		resource := j.Resource()

		userJID := s.JID()
		validFrom = node == userJID.Node() && domain == userJID.Domain()
		if len(resource) > 0 {
			validFrom = validFrom && resource == userJID.Resource()
		}
	}
	return validFrom
}

func (s *c2sStream) isComponentDomain(domain string) bool {
	return false
}

func (s *c2sStream) disconnectWithStreamError(err *streamerror.Error) {
	if s.getState() == connecting {
		s.openStream()
	}
	s.writeElement(err.Element())
	s.disconnectClosingStream(true)
}

func (s *c2sStream) disconnectClosingStream(closeStream bool) {
	if err := s.updateLogoutInfo(); err != nil {
		log.Error(err)
	}
	if presence := s.Presence(); presence != nil && presence.IsAvailable() && s.roster != nil {
		s.roster.BroadcastPresenceAndWait(xml.NewPresence(s.JID(), s.JID(), xml.UnavailableType))
	}
	if closeStream {
		switch s.cfg.Transport.Type {
		case transport.Socket:
			s.tr.WriteString("</stream:stream>")
		case transport.WebSocket:
			s.tr.WriteString(fmt.Sprintf(`<close xmlns="%s" />`, framedStreamNamespace))
		}
	}
	// signal termination...
	s.ctx.Terminate()

	// unregister stream
	if err := c2s.Instance().UnregisterStream(s); err != nil {
		log.Error(err)
	}
	s.setState(disconnected)
	s.tr.Close()
}

func (s *c2sStream) updateLogoutInfo() error {
	var usr *model.User
	var err error
	if presence := s.Presence(); presence != nil {
		if usr, err = storage.Instance().FetchUser(s.Username()); usr != nil && err == nil {
			usr.LoggedOutAt = time.Now()
			if presence.IsUnavailable() {
				usr.LoggedOutStatus = presence.Status()
			}
			return storage.Instance().InsertOrUpdateUser(usr)
		}
	}
	return err
}

func (s *c2sStream) isBlockedJID(jid *xml.JID) bool {
	if jid.IsServer() && c2s.Instance().IsLocalDomain(jid.Domain()) {
		return false
	}
	return c2s.Instance().IsBlockedJID(jid, s.Username())
}

func (s *c2sStream) restart() {
	s.parser = xml.NewParser(s.tr, s.cfg.Transport.MaxStanzaSize)
	s.setState(connecting)
}

func (s *c2sStream) setState(state uint32) {
	atomic.StoreUint32(&s.state, state)
}

func (s *c2sStream) getState() uint32 {
	return atomic.LoadUint32(&s.state)
}
