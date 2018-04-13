/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/server/transport"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/stream/errors"
	"github.com/ortuman/jackal/util"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
)

const (
	connecting uint32 = iota
	connected
	authenticating
	authenticated
	sessionStarted
	disconnected
)

const (
	jabberClientNamespace = "jabber:client"
)

const (
	framedStreamNamespace = "urn:ietf:params:xml:ns:xmpp-framing"
)

const (
	streamNamespace           = "http://etherx.jabber.org/streams"
	tlsNamespace              = "urn:ietf:params:xml:ns:xmpp-tls"
	compressProtocolNamespace = "http://jabber.org/protocol/compress"
	bindNamespace             = "urn:ietf:params:xml:ns:xmpp-bind"
	sessionNamespace          = "urn:ietf:params:xml:ns:xmpp-session"
)

const streamMailboxSize = 32

var (
	errNotExistingAccount = errors.New("account does not exist")
	errResourceNotFound   = errors.New("resource not found")
	errNotAuthenticated   = errors.New("user not authenticated")
)

type c2sContext struct {
	username      string
	domain        string
	resource      string
	jid           *xml.JID
	secured       bool
	authenticated bool
	compressed    bool
	priority      int8
	presence      *xml.Presence
	rosterOnce    uint32
	offlineOnce   uint32
}

type c2sStream struct {
	lock        sync.RWMutex
	cfg         *config.Server
	tr          transport.Transport
	id          string
	connected   uint32
	state       uint32
	ctx         c2sContext
	authrs      []authenticator
	activeAuthr authenticator
	iqHandlers  []module.IQHandler
	roster      *module.ModRoster
	register    *module.XEPRegister
	ping        *module.XEPPing
	offline     *module.ModOffline
	actorCh     chan func()
}

func newStream(id string, tr transport.Transport, cfg *config.Server) *c2sStream {
	s := &c2sStream{
		cfg:     cfg,
		id:      id,
		tr:      tr,
		state:   connecting,
		actorCh: make(chan func(), streamMailboxSize),
	}
	s.ctx.secured = !(cfg.Transport.Type == config.SocketTransportType)

	// assign default domain
	s.ctx.domain = c2s.Instance().DefaultLocalDomain()
	s.ctx.jid, _ = xml.NewJID("", s.ctx.domain, "", true)

	// initialize authenticators
	s.initializeAuthenticators()

	// initialize XEPs
	s.initializeXEPs()

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

// Username returns current stream username.
func (s *c2sStream) Username() string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.ctx.username
}

// Domain returns current stream domain.
func (s *c2sStream) Domain() string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.ctx.domain
}

// Resource returns current stream resource.
func (s *c2sStream) Resource() string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.ctx.resource
}

// JID returns current user JID.
func (s *c2sStream) JID() *xml.JID {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.ctx.jid
}

// Priority returns current presence priority.
func (s *c2sStream) Priority() int8 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.ctx.priority
}

// IsAuthenticated returns whether or not the XMPP stream
// has successfully authenticated.
func (s *c2sStream) IsAuthenticated() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.ctx.authenticated
}

// IsSecured returns whether or not the XMPP stream
// has been secured using SSL/TLS.
func (s *c2sStream) IsSecured() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.ctx.secured
}

// IsCompressed returns whether or not the XMPP stream
// has enabled a compression method.
func (s *c2sStream) IsCompressed() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.ctx.compressed
}

// IsRosterRequested returns whether or not user's roster has been requested.
func (s *c2sStream) IsRosterRequested() bool {
	if s.roster != nil {
		return s.roster.IsRequested()
	}
	return false
}

// Presence returns last sent presence element.
func (s *c2sStream) Presence() *xml.Presence {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.ctx.presence
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

func (s *c2sStream) initializeXEPs() {
	// Roster (https://xmpp.org/rfcs/rfc3921.html#roster)
	s.roster = module.NewRoster(&s.cfg.ModRoster, s)
	s.iqHandlers = append(s.iqHandlers, s.roster)

	// XEP-0012: Last Activity (https://xmpp.org/extensions/xep-0012.html)
	if _, ok := s.cfg.Modules["last_activity"]; ok {
		s.iqHandlers = append(s.iqHandlers, module.NewXEPLastActivity(s))
	}

	// XEP-0030: Service Discovery (https://xmpp.org/extensions/xep-0030.html)
	discoInfo := module.NewXEPDiscoInfo(s)
	s.iqHandlers = append(s.iqHandlers, discoInfo)

	// XEP-0049: Private XML Storage (https://xmpp.org/extensions/xep-0049.html)
	if _, ok := s.cfg.Modules["private"]; ok {
		s.iqHandlers = append(s.iqHandlers, module.NewXEPPrivateStorage(s))
	}

	// XEP-0054: vcard-temp (https://xmpp.org/extensions/xep-0054.html)
	if _, ok := s.cfg.Modules["vcard"]; ok {
		s.iqHandlers = append(s.iqHandlers, module.NewXEPVCard(s))
	}

	// XEP-0077: In-band registration (https://xmpp.org/extensions/xep-0077.html)
	if _, ok := s.cfg.Modules["registration"]; ok {
		s.register = module.NewXEPRegister(&s.cfg.ModRegistration, s)
		s.iqHandlers = append(s.iqHandlers, s.register)
	}

	// XEP-0092: Software Version (https://xmpp.org/extensions/xep-0092.html)
	if _, ok := s.cfg.Modules["version"]; ok {
		s.iqHandlers = append(s.iqHandlers, module.NewXEPVersion(&s.cfg.ModVersion, s))
	}

	// XEP-0199: XMPP Ping (https://xmpp.org/extensions/xep-0199.html)
	if _, ok := s.cfg.Modules["ping"]; ok {
		s.ping = module.NewXEPPing(&s.cfg.ModPing, s)
		s.iqHandlers = append(s.iqHandlers, s.ping)
	}

	// register server disco info identities
	identities := []module.DiscoIdentity{{
		Category: "server",
		Type:     "im",
		Name:     s.cfg.ID,
	}}
	discoInfo.SetIdentities(identities)

	// register disco info features
	var features []string
	for _, iqHandler := range s.iqHandlers {
		features = append(features, iqHandler.AssociatedNamespaces()...)
	}

	// XEP-0160: Offline message storage (https://xmpp.org/extensions/xep-0160.html)
	if _, ok := s.cfg.Modules["offline"]; ok {
		s.offline = module.NewOffline(&s.cfg.ModOffline, s)
		features = append(features, s.offline.AssociatedNamespaces()...)
	}
	discoInfo.SetFeatures(features)
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
	s.lock.Lock()
	s.ctx.domain = elem.To()
	s.lock.Unlock()

	// open stream
	s.openStream()

	features := xml.NewElementName("stream:features")
	features.SetAttribute("xmlns:stream", streamNamespace)
	features.SetAttribute("version", "1.0")

	isSocketTransport := s.cfg.Transport.Type == config.SocketTransportType

	if !s.IsAuthenticated() {
		if isSocketTransport && !s.IsSecured() {
			startTLS := xml.NewElementName("starttls")
			startTLS.SetNamespace("urn:ietf:params:xml:ns:xmpp-tls")
			startTLS.AppendElement(xml.NewElementName("required"))
			features.AppendElement(startTLS)
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
			features.AppendElement(mechanisms)
		}

		// allow In-band registration over encrypted stream only
		allowRegistration := s.IsSecured()

		if _, ok := s.cfg.Modules["registration"]; ok && allowRegistration {
			registerFeature := xml.NewElementNamespace("register", "http://jabber.org/features/iq-register")
			features.AppendElement(registerFeature)
		}
		s.setState(connected)

	} else {
		// attach compression feature
		compressionAvailable := isSocketTransport && s.cfg.Compression.Level != config.NoCompression

		if !s.IsCompressed() && compressionAvailable {
			compression := xml.NewElementNamespace("compression", "http://jabber.org/features/compress")
			method := xml.NewElementName("method")
			method.SetText("zlib")
			compression.AppendElement(method)
			features.AppendElement(compression)
		}
		bind := xml.NewElementNamespace("bind", "urn:ietf:params:xml:ns:xmpp-bind")
		bind.AppendElement(xml.NewElementName("required"))
		features.AppendElement(bind)

		session := xml.NewElementNamespace("session", "urn:ietf:params:xml:ns:xmpp-session")
		features.AppendElement(session)

		if s.roster != nil && s.cfg.ModRoster.Versioning {
			ver := xml.NewElementNamespace("ver", "urn:xmpp:features:rosterver")
			features.AppendElement(ver)
		}
		s.setState(authenticated)
	}
	s.writeElement(features)
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
		stanza, _, err := s.buildStanza(elem, false)
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
		stanza, _, err := s.buildStanza(elem, true)
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

	stanza, toJID, err := s.buildStanza(elem, true)
	if err != nil {
		s.handleElementError(elem, err)
		return
	}
	if s.isComponentDomain(toJID.Domain()) {
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
	s.lock.Lock()
	s.ctx.secured = true
	s.lock.Unlock()

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
	s.lock.Lock()
	s.ctx.compressed = true
	s.lock.Unlock()

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
	s.lock.Lock()
	s.ctx.username = username
	s.ctx.authenticated = true
	s.ctx.jid, _ = xml.NewJID(s.ctx.username, s.ctx.domain, "", true)
	s.lock.Unlock()

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
	if strm := s.userResourceStream(resource); strm != nil {
		switch s.cfg.ResourceConflict {
		case config.Override:
			// override the resource with a server-generated resourcepart...
			h := sha256.New()
			h.Write([]byte(s.ID()))
			resource = hex.EncodeToString(h.Sum(nil))
		case config.Replace:
			// terminate the session of the currently connected client...
			strm.Disconnect(streamerror.ErrResourceConstraint)
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
	s.lock.Lock()
	s.ctx.resource = resource
	s.ctx.jid = userJID
	s.lock.Unlock()

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

	if s.ping != nil {
		s.ping.StartPinging()
	}
	s.setState(sessionStarted)
}

func (s *c2sStream) processStanza(element xml.XElement) {
	switch stanza := element.(type) {
	case *xml.IQ:
		s.processIQ(stanza)
	case *xml.Presence:
		s.processPresence(stanza)
	case *xml.Message:
		s.processMessage(stanza)
	}
}

func (s *c2sStream) processComponentStanza(element xml.XElement) {
}

func (s *c2sStream) processIQ(iq *xml.IQ) {
	if !c2s.Instance().IsLocalDomain(iq.ToJID().Domain()) {
		// TODO(ortuman): Implement XMPP federation
		return
	}

	toJid := iq.ToJID()
	if toJid.IsFull() {
		if err := s.sendElement(iq, toJid); err == errResourceNotFound {
			resp := xml.NewElementFromElement(iq)
			resp.SetFrom(toJid.String())
			resp.SetTo(s.JID().String())
			s.SendElement(resp.ServiceUnavailableError())
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
	if !c2s.Instance().IsLocalDomain(presence.ToJID().Domain()) {
		// TODO(ortuman): Implement XMPP federation
		return
	}
	toJid := presence.ToJID()
	if toJid.IsBare() && (toJid.Node() != s.Username() || toJid.Domain() != s.Domain()) {
		if s.roster != nil {
			s.roster.ProcessPresence(presence)
		}
		return
	}
	if toJid.IsFull() {
		s.sendElement(presence, toJid)
		return
	}

	// set resource priority & availability
	s.lock.Lock()
	s.ctx.priority = presence.Priority()
	s.ctx.presence = presence
	s.lock.Unlock()

	// deliver pending approval notifications
	if s.roster != nil {
		if atomic.CompareAndSwapUint32(&s.ctx.rosterOnce, 0, 1) {
			s.roster.DeliverPendingApprovalNotifications()
			s.roster.ReceivePresences()
		}
		s.roster.BroadcastPresence(presence)
	}

	// deliver offline messages
	if s.offline != nil && s.Priority() >= 0 {
		if atomic.CompareAndSwapUint32(&s.ctx.offlineOnce, 0, 1) {
			s.offline.DeliverOfflineMessages()
		}
	}
}

func (s *c2sStream) processMessage(message *xml.Message) {
	if !c2s.Instance().IsLocalDomain(message.ToJID().Domain()) {
		// TODO(ortuman): Implement XMPP federation
		return
	}
	toJid := message.ToJID()

sendMessage:
	err := s.sendElement(message, toJid)
	switch err {
	case nil:
		break
	case errNotAuthenticated:
		if s.offline != nil {
			if (message.IsChat() || message.IsGroupChat()) && message.IsMessageWithBody() {
				return
			}
			s.offline.ArchiveMessage(message)
		}
	case errResourceNotFound:
		// treat the stanza as if it were addressed to <node@domain>
		toJid = toJid.ToBareJID()
		goto sendMessage
	case errNotExistingAccount:
		response := xml.NewElementFromElement(message)
		response.SetFrom(toJid.String())
		response.SetTo(s.JID().String())
		s.writeElement(response.ServiceUnavailableError())
		return
	default:
		log.Error(err)
	}
}

func (s *c2sStream) restart() {
	s.setState(connecting)
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
	if e, err := s.tr.ReadElement(); e != nil && err == nil {
		s.actorCh <- func() {
			s.readElement(e)
		}
	} else if err != nil {
		if s.getState() == disconnected {
			return // already disconnected...
		}

		var discErr error
		switch err {
		case nil, io.EOF, io.ErrUnexpectedEOF, xml.ErrStreamClosedByPeer:
			break

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
	log.Debugf("RECV: %v", elem)
	s.handleElement(elem)
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
	case config.SocketTransportType:
		ops = xml.NewElementName("stream:stream")
		ops.SetAttribute("xmlns", jabberClientNamespace)
		ops.SetAttribute("xmlns:stream", streamNamespace)
		buf.WriteString(`<?xml version="1.0"?>`)

	case config.WebSocketTransportType:
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

	log.Debugf("SEND: %v", ops)

	s.tr.WriteString(buf.String())
}

func (s *c2sStream) buildStanza(elem xml.XElement, validateFrom bool) (xml.XElement, *xml.JID, error) {
	if err := s.validateNamespace(elem); err != nil {
		return nil, nil, err
	}
	fromJID, toJID, err := s.extractAddresses(elem, true)
	if err != nil {
		return nil, nil, err
	}
	switch elem.Name() {
	case "iq":
		iq, err := xml.NewIQFromElement(elem, fromJID, toJID)
		if err != nil {
			log.Error(err)
			return nil, nil, xml.ErrBadRequest
		}
		return iq, iq.ToJID(), nil

	case "presence":
		presence, err := xml.NewPresenceFromElement(elem, fromJID, toJID)
		if err != nil {
			log.Error(err)
			return nil, nil, xml.ErrBadRequest
		}
		return presence, presence.ToJID(), nil

	case "message":
		message, err := xml.NewMessageFromElement(elem, fromJID, toJID)
		if err != nil {
			log.Error(err)
			return nil, nil, xml.ErrBadRequest
		}
		return message, message.ToJID(), nil
	}
	return nil, nil, streamerror.ErrUnsupportedStanzaType
}

func (s *c2sStream) handleElementError(elem xml.XElement, err error) {
	if streamErr, ok := err.(*streamerror.Error); ok {
		s.disconnectWithStreamError(streamErr)
	} else if stanzaErr, ok := err.(*xml.StanzaError); ok {
		s.writeElement(xml.NewErrorElementFromElement(elem, stanzaErr))
	} else {
		log.Error(err)
	}
}

func (s *c2sStream) validateStreamElement(elem xml.XElement) *streamerror.Error {
	switch s.cfg.Transport.Type {
	case config.SocketTransportType:
		if elem.Name() != "stream:stream" {
			return streamerror.ErrUnsupportedStanzaType
		}
		if elem.Namespace() != jabberClientNamespace || elem.Attributes().Get("xmlns:stream") != streamNamespace {
			return streamerror.ErrInvalidNamespace
		}

	case config.WebSocketTransportType:
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
		toJID, err = xml.NewJID("", s.Domain(), "", true)
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
		case config.SocketTransportType:
			s.tr.WriteString("</stream:stream>")
		case config.WebSocketTransportType:
			s.tr.WriteString(fmt.Sprintf(`<close xmlns="%s" />`, framedStreamNamespace))
		}
	}
	// stop modules
	for _, iqHandler := range s.iqHandlers {
		iqHandler.Done()
	}
	if s.offline != nil {
		s.offline.Done()
	}
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
	if usr, err = storage.Instance().FetchUser(s.Username()); usr != nil && err == nil {
		usr.LoggedOutAt = time.Now()
		if presence := s.Presence(); presence.IsUnavailable() {
			usr.LoggedOutStatus = presence.Status()
		}
		return storage.Instance().InsertOrUpdateUser(usr)
	}
	return err
}

func (s *c2sStream) setState(state uint32) {
	atomic.StoreUint32(&s.state, state)
}

func (s *c2sStream) getState() uint32 {
	return atomic.LoadUint32(&s.state)
}

func (s *c2sStream) userResourceStream(resource string) c2s.Stream {
	strms := c2s.Instance().AvailableStreams(s.Username())
	for _, strm := range strms {
		if strm.Resource() == resource {
			return strm
		}
	}
	return nil
}

func (s *c2sStream) sendElement(element xml.XElement, to *xml.JID) error {
	recipients := c2s.Instance().AvailableStreams(to.Node())
	if len(recipients) == 0 {
		exists, err := storage.Instance().UserExists(to.Node())
		if err != nil {
			return err
		}
		if exists {
			return errNotAuthenticated
		}
		return errNotExistingAccount
	}
	if to.IsFull() {
		for _, strm := range recipients {
			if strm.Resource() == to.Resource() {
				strm.SendElement(element)
				return nil
			}
		}
		return errResourceNotFound
	}
	switch element.(type) {
	case *xml.Message:
		// send to highest priority stream
		strm := recipients[0]
		highestPriority := strm.Priority()
		for i := 1; i < len(recipients); i++ {
			if recipients[i].Priority() > highestPriority {
				strm = recipients[i]
			}
		}
		strm.SendElement(element)

	default:
		// broadcast to all streams
		for _, strm := range recipients {
			strm.SendElement(element)
		}
	}
	return nil
}
