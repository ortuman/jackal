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

package clusterconnmanager

import (
	"context"

	"github.com/jackal-xmpp/stravaganza"
	clusterpb "github.com/ortuman/jackal/pkg/cluster/pb"
	streamqueue "github.com/ortuman/jackal/pkg/module/xep0198/queue"
)

// StreamQueue represents a stream managed queue.
type StreamQueue struct {
	// Elements contains the queue elements.
	Elements []streamqueue.Element

	// Nonce is the nonce queue byte slice.
	Nonce []byte

	// InH is the queue incoming h value.
	InH uint32

	// OutH is the queue outgoing h value.
	OutH uint32
}

// StreamManagement defines a stream management service.
type StreamManagement interface {
	TransferQueue(ctx context.Context, queueID string) (*StreamQueue, error)
}

type streamManagement struct {
	cl clusterpb.StreamManagementClient
}

func (cc *streamManagement) TransferQueue(ctx context.Context, queueID string) (*StreamQueue, error) {
	resp, err := cc.cl.TransferQueue(ctx, &clusterpb.TransferQueueRequest{
		Identifier: queueID,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	elements := make([]streamqueue.Element, 0, len(resp.Elements))

	for _, elem := range resp.Elements {
		b := stravaganza.NewBuilderFromProto(elem.GetStanza())
		stanza, err := b.BuildStanza()
		if err != nil {
			return nil, err
		}
		elements = append(elements, streamqueue.Element{
			Stanza: stanza,
			H:      elem.GetH(),
		})
	}
	return &StreamQueue{
		Elements: elements,
		Nonce:    resp.GetNonce(),
		InH:      resp.GetInH(),
		OutH:     resp.GetOutH(),
	}, nil
}
