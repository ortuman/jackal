// Copyright 2020 The jackal Authors
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

package s2s

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/ortuman/jackal/cluster/kv"
)

func dbKey(dbSecret, sender, target, streamID string) string {
	h := sha256.New()
	h.Write([]byte(dbSecret))
	hm := hmac.New(sha256.New, []byte(hex.EncodeToString(h.Sum(nil))))
	hm.Write([]byte(fmt.Sprintf("%s %s %s", target, sender, streamID)))
	return hex.EncodeToString(hm.Sum(nil))
}

func registerDbRequest(ctx context.Context, sender, target, streamID string, kv kv.KV) error {
	return kv.Put(ctx, dbReqKey(streamID), dbReqVal(sender, target))
}

func unregisterDbRequest(ctx context.Context, streamID string, kv kv.KV) error {
	return kv.Del(ctx, dbReqKey(streamID))
}

func isDbRequestOn(ctx context.Context, sender, target, streamID string, kv kv.KV) (bool, error) {
	val, err := kv.Get(ctx, dbReqKey(streamID))
	if err != nil {
		return false, err
	}
	if len(val) == 0 {
		return false, nil
	}
	var s0, t0 string
	_, _ = fmt.Sscanf(string(val), "%s %s", &s0, &t0)
	return s0 == sender && t0 == target, nil
}

func dbReqKey(streamID string) string {
	return fmt.Sprintf("db://%s", streamID)
}

func dbReqVal(sender, target string) string {
	return fmt.Sprintf("%s %s", sender, target)
}
