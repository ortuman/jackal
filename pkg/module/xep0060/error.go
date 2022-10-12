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
	"errors"

	"github.com/jackal-xmpp/stravaganza"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
)

var (
	errFeatureNotSupported     = errors.New("feature not supported")
	errNodeIDRequired          = errors.New("node id required")
	errNodeNotFound            = errors.New("node does not exist")
	errInsufficientPrivileges  = errors.New("insufficient privileges")
	errFeatureNotImplemented   = errors.New("feature not implemented")
	errConfigRetrievalDisabled = errors.New("configuration retrieval disabled")
)

type serviceError struct {
	featureName   string
	underlyingErr error
}

func newServiceError(underlyingErr error) *serviceError {
	return &serviceError{underlyingErr: underlyingErr}
}

func newServiceErrorWithFeatureName(underlyingErr error, featureName string) *serviceError {
	return &serviceError{featureName: featureName, underlyingErr: underlyingErr}
}

func (e *serviceError) Error() string {
	return e.underlyingErr.Error()
}

func unsupportedStanzaError(iq *stravaganza.IQ, reason stanzaerror.Reason, feature string) stravaganza.Stanza {
	unsupportedElem := stravaganza.NewBuilder("unsupported").
		WithAttribute(stravaganza.Namespace, errorNS()).
		WithAttribute("feature", feature).
		Build()

	return xmpputil.MakeErrorStanzaWithApplicationElement(iq,
		unsupportedElem,
		reason,
	)
}

func nodeIDRequiredError(iq *stravaganza.IQ, reason stanzaerror.Reason) stravaganza.Stanza {
	nodeIDRequired := stravaganza.NewBuilder("nodeid-required").
		WithAttribute(stravaganza.Namespace, errorNS()).
		Build()

	return xmpputil.MakeErrorStanzaWithApplicationElement(iq,
		nodeIDRequired,
		reason,
	)
}
