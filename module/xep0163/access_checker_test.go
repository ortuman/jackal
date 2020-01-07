/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0163

import (
	"context"
	"testing"

	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	rostermodel "github.com/ortuman/jackal/model/roster"
	memorystorage "github.com/ortuman/jackal/storage/memory"
	"github.com/stretchr/testify/require"
)

func TestAccessChecker_Open(t *testing.T) {
	ac := &accessChecker{
		host:        "ortuman@jackal.im",
		nodeID:      "princely_musings",
		accessModel: pubsubmodel.Open,
		rosterRep:   memorystorage.NewRoster(),
	}

	err := ac.checkAccess(context.Background(), "noelia@jackal.im")
	require.Nil(t, err)
}

func TestAccessChecker_Outcast(t *testing.T) {
	ac := &accessChecker{
		host:        "ortuman@jackal.im",
		nodeID:      "princely_musings",
		accessModel: pubsubmodel.Open,
		affiliation: &pubsubmodel.Affiliation{JID: "noelia@jackal.im", Affiliation: pubsubmodel.Outcast},
		rosterRep:   memorystorage.NewRoster(),
	}

	err := ac.checkAccess(context.Background(), "noelia@jackal.im")
	require.NotNil(t, err)
	require.Equal(t, errOutcastMember, err)
}

func TestAccessChecker_PresenceSubscription(t *testing.T) {
	rosterRep := memorystorage.NewRoster()
	ac := &accessChecker{
		host:        "ortuman@jackal.im",
		nodeID:      "princely_musings",
		accessModel: pubsubmodel.Presence,
		rosterRep:   rosterRep,
	}

	err := ac.checkAccess(context.Background(), "noelia@jackal.im")
	require.NotNil(t, err)
	require.Equal(t, errPresenceSubscriptionRequired, err)

	_, _ = rosterRep.UpsertRosterItem(context.Background(), &rostermodel.Item{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Subscription: rostermodel.SubscriptionFrom,
	})

	err = ac.checkAccess(context.Background(), "noelia@jackal.im")
	require.Nil(t, err)
}

func TestAccessChecker_RosterGroup(t *testing.T) {
	rosterRep := memorystorage.NewRoster()
	ac := &accessChecker{
		host:                "ortuman@jackal.im",
		nodeID:              "princely_musings",
		rosterAllowedGroups: []string{"Family"},
		accessModel:         pubsubmodel.Roster,
		rosterRep:           rosterRep,
	}

	err := ac.checkAccess(context.Background(), "noelia@jackal.im")
	require.NotNil(t, err)
	require.Equal(t, errNotInRosterGroup, err)

	_, _ = rosterRep.UpsertRosterItem(context.Background(), &rostermodel.Item{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Groups:       []string{"Family"},
		Subscription: rostermodel.SubscriptionFrom,
	})

	err = ac.checkAccess(context.Background(), "noelia@jackal.im")
	require.Nil(t, err)
}

func TestAccessChecker_Member(t *testing.T) {
	ac := &accessChecker{
		host:        "ortuman@jackal.im",
		nodeID:      "princely_musings",
		accessModel: pubsubmodel.WhiteList,
		affiliation: &pubsubmodel.Affiliation{JID: "noelia@jackal.im", Affiliation: pubsubmodel.Member},
		rosterRep:   memorystorage.NewRoster(),
	}

	err := ac.checkAccess(context.Background(), "noelia2@jackal.im")
	require.NotNil(t, err)
	require.Equal(t, errNotOnWhiteList, err)

	err = ac.checkAccess(context.Background(), "noelia@jackal.im")
	require.Nil(t, err)
}
