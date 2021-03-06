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

package c2s

import (
	"strconv"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	c2smodel "github.com/ortuman/jackal/pkg/model/c2s"
)

func testMessageStanza() *stravaganza.Message {
	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("from", "noelia@jackal.im/yard")
	b.WithAttribute("to", "ortuman@jackal.im/balcony")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText("I'll give thee a wind.").
			Build(),
	)
	msg, _ := b.BuildMessage()
	return msg
}

func testResource(instanceID string, priority int8) c2smodel.Resource {
	pr, _ := stravaganza.NewPresenceBuilder().
		WithAttribute(stravaganza.From, "ortuman@jackal.im/yard").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithChild(
			stravaganza.NewBuilder("priority").
				WithText(strconv.Itoa(int(priority))).
				Build(),
		).
		BuildPresence()

	jd, _ := jid.New("ortuman", "jackal.im", "yard", true)
	return c2smodel.Resource{
		InstanceID: instanceID,
		JID:        jd,
		Presence:   pr,
		Info:       c2smodel.Info{M: map[string]string{"k1": "v1", "k2": "v2"}},
	}
}
