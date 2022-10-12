// Copyright 2022 The jackal Authors
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

package xep0060

import (
	"context"

	kitlog "github.com/go-kit/log"
	"github.com/jackal-xmpp/stravaganza"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/storage/repository"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
)

// Service represents a PubSub service.
type Service struct {
	cfg    ServiceConfig
	router router.Router
	rep    repository.Repository
	logger kitlog.Logger
}

// NewService returns a new PubSub service instance.
func NewService(
	cfg ServiceConfig,
	router router.Router,
	rep repository.Repository,
	logger kitlog.Logger,
) *Service {
	return &Service{
		cfg:    cfg,
		router: router,
		rep:    rep,
		logger: logger,
	}
}

// ProcessIQ processes a PubSub IQ.
func (m *Service) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	pubSubElement := iq.Child("pubsub")
	if pubSubElement == nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
	var err error
	switch pubSubElement.Attribute(stravaganza.Namespace) {
	case pubSubNamespace:
		err = m.processRequest(ctx, iq, pubSubElement)

	case pubSubOwnerNS():
		err = m.processOwnerRequest(ctx, iq, pubSubElement)
	}
	return m.handleError(ctx, iq, err)
}

func (m *Service) processRequest(ctx context.Context, iq *stravaganza.IQ, pubSubElement stravaganza.Element) error {
	if create := pubSubElement.Child("create"); create != nil && iq.IsSet() {
		return m.createNode(ctx, iq, create, pubSubElement.Child("configure"))
	}
	return newServiceError(errFeatureNotImplemented)
}

func (m *Service) processOwnerRequest(ctx context.Context, iq *stravaganza.IQ, pubSubElement stravaganza.Element) error {
	if configure := pubSubElement.Child("configure"); configure != nil {
		if iq.IsGet() {
			return m.getNodeConfig(ctx, iq, configure)
		} else if iq.IsSet() {
			return m.setNodeConfig(ctx, iq, configure)
		}
	} else if pubSubElement.Child("default") != nil && iq.IsGet() {
		return m.getDefaultConfig(ctx, iq)
	} else if del := pubSubElement.Child("delete"); del != nil && iq.IsSet() {
		return m.deleteNode(ctx, iq, del)
	} else if purge := pubSubElement.Child("purge"); purge != nil && iq.IsSet() {
		return m.purgeNode(ctx, iq, purge)
	}
	return newServiceError(errFeatureNotImplemented)
}

func (m *Service) handleError(ctx context.Context, iq *stravaganza.IQ, err error) error {
	if err == nil {
		return nil
	}
	switch e := err.(type) {
	case *serviceError:
		m.handleServiceError(ctx, iq, e)
		return nil

	default:
		switch e {
		case ErrNotRegisted:
			_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.RegistrationRequired))
			return nil

		case ErrNotAllowed:
			_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.Forbidden))
			return nil

		default:
			_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
			return err
		}
	}
}

func (m *Service) handleServiceError(ctx context.Context, iq *stravaganza.IQ, serviceErr *serviceError) {
	switch serviceErr.underlyingErr {
	case errFeatureNotSupported, errConfigRetrievalDisabled:
		_, _ = m.router.Route(ctx, unsupportedStanzaError(iq, stanzaerror.FeatureNotImplemented, serviceErr.featureName))

	case errNodeIDRequired:
		_, _ = m.router.Route(ctx, nodeIDRequiredError(iq, stanzaerror.BadRequest))

	case errNodeNotFound:
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.ItemNotFound))

	case errInsufficientPrivileges:
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.Forbidden))

	case errFeatureNotImplemented:
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.FeatureNotImplemented))
	}
}
