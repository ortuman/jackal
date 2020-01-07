/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0163

import (
	"context"
	"errors"
	"fmt"

	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/xmpp/jid"
)

var (
	errOutcastMember                = errors.New("pep: outcast member")
	errPresenceSubscriptionRequired = errors.New("pep: presence subscription required")
	errNotInRosterGroup             = errors.New("pep: not in roster group")
	errNotOnWhiteList               = errors.New("pep: not on whitelist")
)

type accessChecker struct {
	host                string
	nodeID              string
	accessModel         string
	rosterAllowedGroups []string
	affiliation         *pubsubmodel.Affiliation
	rosterRep           repository.Roster
}

func (ac *accessChecker) checkAccess(ctx context.Context, j string) error {
	aff := ac.affiliation
	if aff != nil && aff.Affiliation == pubsubmodel.Outcast {
		return errOutcastMember
	}
	switch ac.accessModel {
	case pubsubmodel.Open:
		return nil

	case pubsubmodel.Presence:
		allowed, err := ac.checkPresenceAccess(ctx, j)
		if err != nil {
			return err
		}
		if !allowed {
			return errPresenceSubscriptionRequired
		}

	case pubsubmodel.Roster:
		allowed, err := ac.checkRosterAccess(ctx, j)
		if err != nil {
			return err
		}
		if !allowed {
			return errNotInRosterGroup
		}

	case pubsubmodel.WhiteList:
		allowed, err := ac.checkWhitelistAccess(j)
		if err != nil {
			return err
		}
		if !allowed {
			return errNotOnWhiteList
		}

	default:
		return fmt.Errorf("pep: unrecognized access model: %s", ac.accessModel)
	}
	return nil
}

func (ac *accessChecker) checkPresenceAccess(ctx context.Context, j string) (bool, error) {
	userJID, _ := jid.NewWithString(ac.host, true)
	contactJID, _ := jid.NewWithString(j, true)

	ri, err := ac.rosterRep.FetchRosterItem(ctx, userJID.Node(), contactJID.ToBareJID().String())
	if err != nil {
		return false, err
	}
	allowed := ri != nil && (ri.Subscription == rostermodel.SubscriptionFrom || ri.Subscription == rostermodel.SubscriptionBoth)
	return allowed, nil
}

func (ac *accessChecker) checkRosterAccess(ctx context.Context, j string) (bool, error) {
	userJID, _ := jid.NewWithString(ac.host, true)
	contactJID, _ := jid.NewWithString(j, true)

	ri, err := ac.rosterRep.FetchRosterItem(ctx, userJID.Node(), contactJID.ToBareJID().String())
	if err != nil {
		return false, err
	}
	if ri == nil {
		return false, nil
	}
	for _, group := range ri.Groups {
		for _, allowedGroup := range ac.rosterAllowedGroups {
			if group == allowedGroup {
				return true, nil
			}
		}
	}
	return false, nil
}

func (ac *accessChecker) checkWhitelistAccess(j string) (bool, error) {
	aff := ac.affiliation
	if aff == nil || j != aff.JID {
		return false, nil
	}
	return aff.Affiliation == pubsubmodel.Owner || aff.Affiliation == pubsubmodel.Member, nil
}
