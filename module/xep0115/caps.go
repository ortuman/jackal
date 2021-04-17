// Copyright 2021 The jackal Authors
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

package xep0115

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/event"
	"github.com/ortuman/jackal/log"
	capsmodel "github.com/ortuman/jackal/model/caps"
	discomodel "github.com/ortuman/jackal/model/disco"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/repository"
	"github.com/ortuman/jackal/router"
)

const (
	capabilitiesFeature = "http://jabber.org/protocol/caps"

	discoInfoNamespace = "http://jabber.org/protocol/disco#info"
	formsNamespace     = "jabber:x:data"
)

var (
	errIdentityRepeated      = errors.New("xep0115: more than one service discovery identity with the same category/type/lang/name")
	errFeatureRepeated       = errors.New("xep0115: more than one service discovery feature with the same XML character data")
	errFormTypeNotHidden     = errors.New("xep0115: FORM_TYPE field is not of type \"hidden\"")
	errFormTypeFieldNotFound = errors.New("xep0115: FORM_TYPE field not found")
	errFormTypeFieldBadValue = errors.New("xep0115: FORM_TYPE field contains more than one <value/> element")
	errFormTypeRepeated      = errors.New("xep0115: more than one extended service discovery information form with the same FORM_TYPE")
)

var hashFn = map[string]func() hash.Hash{
	"sha-1":   sha1.New,
	"sha-224": sha256.New224,
	"sha-256": sha256.New,
	"sha-384": sha512.New384,
	"sha-512": sha512.New,
}

type capsInfo struct {
	hash string
	node string
	ver  string
}

const (
	// ModuleName represents entity capabilities module name.
	ModuleName = "caps"

	// XEPNumber represents entity capabilities XEP number.
	XEPNumber = "0115"
)

// Capabilities represents entity capabilities (XEP-0115) module type.
type Capabilities struct {
	disco  *xep0030.Disco
	router router.Router
	rep    repository.Capabilities
	sn     *sonar.Sonar
	subs   []sonar.SubID

	mu     sync.RWMutex
	reqs   map[string]capsInfo
	clrTms map[string]*time.Timer
}

// New creates and initializes a new Capabilities instance.
func New(
	disco *xep0030.Disco,
	router router.Router,
	rep repository.Capabilities,
	sn *sonar.Sonar,
) *Capabilities {
	return &Capabilities{
		disco:  disco,
		router: router,
		rep:    rep,
		sn:     sn,
		reqs:   make(map[string]capsInfo),
		clrTms: make(map[string]*time.Timer),
	}
}

// Name returns entity capabilities module name.
func (m *Capabilities) Name() string { return ModuleName }

// StreamFeature returns entity capabilities module stream feature.
func (m *Capabilities) StreamFeature(ctx context.Context, domain string) (stravaganza.Element, error) {
	if m.disco == nil {
		return nil, nil
	}
	jd, _ := jid.NewWithString(domain, true)

	srvProv := m.disco.ServerProvider()
	identities := srvProv.Identities(ctx, jd, jd, "")
	features, _ := srvProv.Features(ctx, jd, jd, "")
	forms, _ := srvProv.Forms(ctx, jd, jd, "")

	ver := computeVer(identities, features, forms, sha256.New)
	return stravaganza.NewBuilder("c").
		WithAttribute(stravaganza.Namespace, capabilitiesFeature).
		WithAttribute("hash", "sha-256").
		WithAttribute("node", fmt.Sprintf("http://%s", domain)).
		WithAttribute("ver", ver).
		Build(), nil
}

// ServerFeatures returns entity capabilities module server disco features.
func (m *Capabilities) ServerFeatures(_ context.Context) ([]string, error) {
	return nil, nil
}

// AccountFeatures returns entity capabilities module account disco features.
func (m *Capabilities) AccountFeatures(_ context.Context) ([]string, error) {
	return nil, nil
}

// Start starts entity capabilities module.
func (m *Capabilities) Start(_ context.Context) error {
	m.subs = append(m.subs, m.sn.Subscribe(event.C2SStreamPresenceReceived, m.onC2SPresenceRecv))
	m.subs = append(m.subs, m.sn.Subscribe(event.S2SInStreamPresenceReceived, m.onS2SPresenceRecv))
	m.subs = append(m.subs, m.sn.Subscribe(event.C2SStreamIQReceived, m.onC2SIQRecv))
	m.subs = append(m.subs, m.sn.Subscribe(event.S2SInStreamIQReceived, m.onS2SIQRecv))

	log.Infow("Started capabilities module", "xep", XEPNumber)
	return nil
}

// Stop stops entity capabilities module.
func (m *Capabilities) Stop(_ context.Context) error {
	for _, sub := range m.subs {
		m.sn.Unsubscribe(sub)
	}
	log.Infow("Stopped capabilities module", "xep", XEPNumber)
	return nil
}

func (m *Capabilities) onC2SPresenceRecv(ctx context.Context, ev sonar.Event) error {
	inf := ev.Info().(*event.C2SStreamEventInfo)
	pr := inf.Stanza.(*stravaganza.Presence)
	return m.processPresence(ctx, pr)
}

func (m *Capabilities) onS2SPresenceRecv(ctx context.Context, ev sonar.Event) error {
	inf := ev.Info().(*event.S2SStreamEventInfo)
	pr := inf.Stanza.(*stravaganza.Presence)
	return m.processPresence(ctx, pr)
}

func (m *Capabilities) onC2SIQRecv(ctx context.Context, ev sonar.Event) error {
	inf := ev.Info().(*event.C2SStreamEventInfo)
	iq := inf.Stanza.(*stravaganza.IQ)
	return m.processIQ(ctx, iq)
}

func (m *Capabilities) onS2SIQRecv(ctx context.Context, ev sonar.Event) error {
	inf := ev.Info().(*event.S2SStreamEventInfo)
	iq := inf.Stanza.(*stravaganza.IQ)
	return m.processIQ(ctx, iq)
}

func (m *Capabilities) processPresence(ctx context.Context, pr *stravaganza.Presence) error {
	if pr.ToJID().IsFull() {
		return nil
	}
	caps := pr.ChildNamespace("c", capabilitiesFeature)
	if caps == nil {
		return nil
	}
	h := caps.Attribute("hash")
	if hashFn[h] == nil {
		log.Warnw(fmt.Sprintf("Unrecognized hashing algorithm: %s", h), "xep", XEPNumber)
		return nil
	}
	ci := capsInfo{
		hash: h,
		node: caps.Attribute("node"),
		ver:  caps.Attribute("ver"),
	}
	// fetch registered capabilities
	exist, err := m.rep.CapabilitiesExist(ctx, ci.node, ci.ver)
	if err != nil {
		return err
	}
	if exist {
		return nil
	}
	m.requestDiscoInfo(ctx, pr.FromJID(), pr.ToJID(), ci)
	return nil
}

func (m *Capabilities) processIQ(ctx context.Context, iq *stravaganza.IQ) error {
	reqID := iq.Attribute(stravaganza.ID)

	m.mu.Lock()
	if tm := m.clrTms[reqID]; tm != nil {
		tm.Stop()
	}
	nv, ok := m.reqs[reqID]
	if !ok {
		m.mu.Unlock()
		return nil
	}
	delete(m.reqs, reqID)
	delete(m.clrTms, reqID)
	m.mu.Unlock()

	if err := m.processDiscoInfo(ctx, iq, nv); err != nil {
		log.Warnw(fmt.Sprintf("Failed to verify disco info: %v", err), "xep", XEPNumber)
	}
	return nil
}

func (m *Capabilities) requestDiscoInfo(ctx context.Context, fromJID, toJID *jid.JID, ci capsInfo) {
	reqID := uuid.New().String()

	m.mu.Lock()
	m.reqs[reqID] = ci
	m.clrTms[reqID] = time.AfterFunc(time.Minute, func() {
		m.clearPendingReq(reqID) // discard pending request
	})
	m.mu.Unlock()

	discoIQ, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, reqID).
		WithAttribute(stravaganza.From, toJID.String()).
		WithAttribute(stravaganza.To, fromJID.String()).
		WithAttribute(stravaganza.Type, stravaganza.GetType).
		WithChild(
			stravaganza.NewBuilder("query").
				WithAttribute(stravaganza.Namespace, discoInfoNamespace).
				WithAttribute("node", fmt.Sprintf("%s#%s", ci.node, ci.ver)).
				Build(),
		).
		BuildIQ()

	_, _ = m.router.Route(ctx, discoIQ)
}

func (m *Capabilities) processDiscoInfo(ctx context.Context, iq *stravaganza.IQ, ci capsInfo) error {
	dq := iq.ChildNamespace("query", discoInfoNamespace)
	if dq == nil || !iq.IsResult() {
		return nil
	}
	var identities []discomodel.Identity
	var features []discomodel.Feature
	var forms []xep0004.DataForm

	// get identities
	for _, idnEl := range dq.Children("identity") {
		identities = append(identities, discomodel.Identity{
			Category: idnEl.Attribute("category"),
			Name:     idnEl.Attribute("name"),
			Type:     idnEl.Attribute("type"),
			Lang:     idnEl.Attribute(stravaganza.Language),
		})
	}
	// get features
	for _, featureEl := range dq.Children("feature") {
		features = append(features, featureEl.Attribute("var"))
	}
	// get forms
	for _, formEl := range dq.ChildrenNamespace("x", formsNamespace) {
		form, err := xep0004.NewFormFromElement(formEl)
		if err != nil {
			return err
		}
		forms = append(forms, *form)
	}
	// validate discovery information response
	// (https://xmpp.org/extensions/xep-0115.html#ver-proc)
	if err := validateIdentities(identities); err != nil {
		return err
	}
	if err := validateFeatures(features); err != nil {
		return err
	}
	if err := validateForms(forms); err != nil {
		return err
	}
	// compute verification and store capabilities entity if matches previous received hash
	ver := computeVer(identities, features, forms, hashFn[ci.hash])
	if ver != ci.ver {
		return fmt.Errorf("xep0115: verification string mismatch: got %s, expected %s", ver, ci.ver)
	}
	err := m.rep.UpsertCapabilities(ctx, &capsmodel.Capabilities{
		Node:     ci.node,
		Ver:      ci.ver,
		Features: features,
	})
	if err != nil {
		return err
	}
	log.Infow("Entity capabilities globally cached", "node", ci.node, "ver", ci.ver, "xep", XEPNumber)
	return nil
}

func (m *Capabilities) clearPendingReq(reqID string) {
	m.mu.Lock()
	delete(m.reqs, reqID)
	delete(m.clrTms, reqID)
	m.mu.Unlock()
}

func validateIdentities(identities []discomodel.Identity) error {
	ids := make(map[string]int, len(identities))
	for _, identity := range identities {
		s := fmt.Sprintf("%s/%s/%s/%s", identity.Category, identity.Type, identity.Lang, identity.Name)
		ids[s] = ids[s] + 1
	}
	for _, cnt := range ids {
		if cnt > 1 {
			return errIdentityRepeated
		}
	}
	return nil
}

func validateFeatures(features []discomodel.Feature) error {
	fs := make(map[string]int, len(features))
	for _, f := range features {
		fs[f] = fs[f] + 1
	}
	for _, cnt := range fs {
		if cnt > 1 {
			return errFeatureRepeated
		}
	}
	return nil
}

func validateForms(forms []xep0004.DataForm) error {
	fts := make(map[string]int, len(forms)) // keep track of FORM_TYPE values
	for _, form := range forms {
		for _, f := range form.Fields {
			if f.Var == xep0004.FormType {
				if f.Type != xep0004.Hidden {
					return errFormTypeNotHidden
				}
				if len(f.Values) != 1 {
					return errFormTypeFieldBadValue
				}
				v := f.Values[0]
				fts[v] = fts[v] + 1
			}
		}
	}
	if len(fts) != len(forms) {
		return errFormTypeFieldNotFound
	}
	for _, cnt := range fts {
		if cnt > 1 {
			return errFormTypeRepeated
		}
	}
	return nil
}

func computeVer(
	identities []discomodel.Identity,
	features []discomodel.Feature,
	forms []xep0004.DataForm,
	hFn func() hash.Hash,
) string {
	var sb strings.Builder

	// sort the service discovery identities
	sort.Slice(identities, func(i, j int) bool {
		if identities[i].Category != identities[j].Category {
			return identities[i].Category < identities[j].Category
		}
		if identities[i].Type != identities[j].Type {
			return identities[i].Type < identities[j].Type
		}
		return identities[i].Lang < identities[j].Lang
	})
	for _, identity := range identities {
		sb.WriteString(fmt.Sprintf("%s/%s/%s/%s<", identity.Category, identity.Type, identity.Lang, identity.Name))
	}
	// sort disco info features
	sort.Slice(features, func(i, j int) bool {
		return features[i] < features[j]
	})
	for _, f := range features {
		sb.WriteString(fmt.Sprintf("%s<", f))
	}
	// sort extended info forms
	sort.Slice(forms, func(i, j int) bool {
		return getFormTypeValue(&forms[i]) < getFormTypeValue(&forms[j])
	})
	for _, form := range forms {
		sb.WriteString(getFormTypeValue(&form))
		sb.WriteString("<")

		// get rid of FORM_TYPE field and sort resulting fields
		fields := filterFormTypeField(form.Fields)
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].Var < fields[j].Var
		})
		for _, field := range fields {
			sb.WriteString(field.Var)
			sb.WriteString("<")

			// sort values
			var values []string
			for _, val := range field.Values {
				values = append(values, val)
			}
			sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })

			for _, val := range values {
				sb.WriteString(val)
				sb.WriteString("<")
			}
		}
	}
	h := hFn()
	_, _ = h.Write([]byte(sb.String()))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func getFormTypeValue(form *xep0004.DataForm) string {
	for _, f := range form.Fields {
		if f.Var == xep0004.FormType {
			return f.Values[0]
		}
	}
	return ""
}

func filterFormTypeField(fields []xep0004.Field) []xep0004.Field {
	var f []xep0004.Field
	for _, field := range fields {
		if field.Var == xep0004.FormType {
			continue
		}
		f = append(f, field)
	}
	return f
}
