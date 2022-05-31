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

package clusterserver

import (
	"context"

	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	"github.com/ortuman/jackal/pkg/cluster/pb"
	streamqueue "github.com/ortuman/jackal/pkg/module/xep0198/queue"
)

type streamManagementService struct {
	pb.UnimplementedStreamManagementServer
	stmQueueMap *streamqueue.QueueMap
}

func newStreamManagementService(stmQueueMap *streamqueue.QueueMap) *streamManagementService {
	return &streamManagementService{stmQueueMap: stmQueueMap}
}

func (s *streamManagementService) TransferQueue(_ context.Context, req *pb.TransferQueueRequest) (*pb.TransferQueueResponse, error) {
	if s.stmQueueMap == nil {
		return nil, nil // xep0198 not enabled
	}
	sq := s.stmQueueMap.Delete(req.Identifier)
	if sq == nil {
		return nil, nil
	}
	// first, let's disconnect hibernated c2s stream
	if err := <-sq.GetStream().Disconnect(streamerror.E(streamerror.Conflict)); err != nil {
		return nil, err
	}
	// transfer stream queue
	var resp pb.TransferQueueResponse

	elements := sq.Elements()
	for _, elem := range elements {
		resp.Elements = append(resp.Elements, &pb.QueueElement{
			Stanza: elem.Stanza.Proto(),
			H:      elem.H,
		})
	}
	resp.Nonce = sq.Nonce()
	resp.InH = sq.InboundH()
	resp.OutH = sq.OutboundH()

	return &resp, nil
}
