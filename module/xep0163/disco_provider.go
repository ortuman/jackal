package xep0163

import (
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

type discoInfoProvider struct {
	host string
}

func (p *discoInfoProvider) Identities(_, _ *jid.JID, _ string) []xep0030.Identity { return nil }

func (p *discoInfoProvider) Features(_, _ *jid.JID, _ string) ([]xep0030.Feature, *xmpp.StanzaError) {
	return nil, nil
}

func (p *discoInfoProvider) Form(_, _ *jid.JID, _ string) (*xep0004.DataForm, *xmpp.StanzaError) {
	return nil, nil
}

func (p *discoInfoProvider) Items(toJID, fromJID *jid.JID, node string) ([]xep0030.Item, *xmpp.StanzaError) {
	return nil, nil
}
