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

package pubsubmodel

// NodeAccessModel2String defines an NodeAccessModel to string map.
var NodeAccessModel2String = map[NodeAccessModel]string{
	NodeAccessModel_NAM_UNKNOWN:   "unknown",
	NodeAccessModel_NAM_AUTHORIZE: "authorize",
	NodeAccessModel_NAM_OPEN:      "open",
	NodeAccessModel_NAM_PRESENCE:  "presence",
	NodeAccessModel_NAM_ROSTER:    "roster",
	NodeAccessModel_NAM_WHITELIST: "whitelist",
}

// String2NodeAccessModel defines a strign to NodeAccessModel map.
var String2NodeAccessModel = map[string]NodeAccessModel{
	"unknown":   NodeAccessModel_NAM_UNKNOWN,
	"authorize": NodeAccessModel_NAM_AUTHORIZE,
	"open":      NodeAccessModel_NAM_OPEN,
	"presence":  NodeAccessModel_NAM_PRESENCE,
	"roster":    NodeAccessModel_NAM_ROSTER,
	"whitelist": NodeAccessModel_NAM_WHITELIST,
}

// Affiliation2String defines an AffiliationState to string map.
var Affiliation2String = map[AffiliationState]string{
	AffiliationState_AFF_NONE:         "none",
	AffiliationState_AFF_OWNER:        "owner",
	AffiliationState_AFF_PUBLISHER:    "publisher",
	AffiliationState_AFF_PUBLISH_ONLY: "publish_only",
	AffiliationState_AFF_MEMBER:       "member",
	AffiliationState_AFF_OUTCAST:      "outcast",
}

// String2Affiliation defines a strign to AffiliationState map.
var String2Affiliation = map[string]AffiliationState{
	"none":         AffiliationState_AFF_NONE,
	"owner":        AffiliationState_AFF_OWNER,
	"publisher":    AffiliationState_AFF_PUBLISHER,
	"publish_only": AffiliationState_AFF_PUBLISH_ONLY,
	"member":       AffiliationState_AFF_MEMBER,
	"outcast":      AffiliationState_AFF_OUTCAST,
}

// Subscription2String defines an SubscriptionState to string map.
var Subscription2String = map[SubscriptionState]string{
	SubscriptionState_SUB_NONE:         "none",
	SubscriptionState_SUB_PENDING:      "pending",
	SubscriptionState_SUB_SUBSCRIBED:   "subscribed",
	SubscriptionState_SUB_UNCONFIGURED: "unconfigured",
}

// String2Subscription defines a strign to SubscriptionState map.
var String2Subscription = map[string]SubscriptionState{
	"none":         SubscriptionState_SUB_NONE,
	"pending":      SubscriptionState_SUB_PENDING,
	"subscribed":   SubscriptionState_SUB_SUBSCRIBED,
	"unconfigured": SubscriptionState_SUB_UNCONFIGURED,
}
