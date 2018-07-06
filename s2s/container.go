/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"sync"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/stream"
)

var inContainer inMap

type inMap struct{ m sync.Map }

func (m *inMap) set(stm stream.S2SIn) {
	m.m.Store(stm.ID(), stm)
	log.Infof("registered s2s in stream... (id: %s)", stm.ID())
}

func (m *inMap) delete(stm stream.S2SIn) {
	m.m.Delete(stm.ID())
	log.Infof("unregistered s2s in stream... (id: %s)", stm.ID())
}

var outContainer outMap

type outMap struct{ m sync.Map }

func (m *outMap) getOrDial(localDomain, remoteDomain string) (stream.S2SOut, error) {
	domainPair := localDomain + ":" + remoteDomain
	s, loaded := m.m.LoadOrStore(domainPair, newOutStream())
	if !loaded {
		outCfg, err := defaultDialer.dial(localDomain, remoteDomain)
		if err != nil {
			log.Error(err)
			m.m.Delete(domainPair)
			return nil, err
		}
		s.(*outStream).start(outCfg)
		log.Infof("registered s2s out stream... (domainpair: %s)", domainPair)
	}
	return s.(*outStream), nil
}

func (m *outMap) delete(stm stream.S2SOut) {
	domainPair := stm.ID()
	m.m.Delete(domainPair)
	log.Infof("unregistered s2s out stream... (domainpair: %s)", domainPair)
}
