package s2s

import (
	"crypto/tls"
	"time"

	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/xmpp"
)

type streamConfig struct {
	keyGen          *keyGen
	localDomain     string
	remoteDomain    string
	connectTimeout  time.Duration
	timeout         time.Duration
	tls             *tls.Config
	transport       transport.Transport
	maxStanzaSize   int
	dbVerify        xmpp.XElement
	dialer          *dialer
	onInDisconnect  func(s stream.S2SIn)
	onOutDisconnect func(s stream.S2SOut)
}
