/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0163

import (
	"errors"
	"fmt"

	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/xmpp/jid"
)

var (
	errPresenceSubscriptionRequired = errors.New("pep: presence subscription required")
	errNotInRosterGroup             = errors.New("pep: not in roster group")
	errNotOnWhiteList               = errors.New("pep: not on whitelist")
)

type accessChecker struct {
	accessModel         string
	rosterAllowedGroups []string
	affiliations        []pubsubmodel.Affiliation
}

func (ac *accessChecker) checkAccess(host, j string) error {
	switch ac.accessModel {
	case pubsubmodel.Open:
		return nil

	case pubsubmodel.Presence:
		allowed, err := ac.checkPresenceAccess(host, j)
		if err != nil {
			return err
		}
		if !allowed {
			return errPresenceSubscriptionRequired
		}

	case pubsubmodel.Roster:
		allowed, err := ac.checkRosterAccess(host, j)
		if err != nil {
			return err
		}
		if !allowed {
			return errNotInRosterGroup
		}

	case pubsubmodel.WhiteList:
		if !ac.checkWhitelistAccess(j) {
			return errNotOnWhiteList
		}

	default:
		return fmt.Errorf("pep: unrecognized access model: %s", ac.accessModel)
	}
	return nil
}

func (ac *accessChecker) checkPresenceAccess(host, j string) (bool, error) {
	userJID, _ := jid.NewWithString(host, true)
	contactJID, _ := jid.NewWithString(j, true)

	ri, err := storage.FetchRosterItem(userJID.Node(), contactJID.ToBareJID().String())
	if err != nil {
		return false, err
	}
	allowed := ri != nil && (ri.Subscription == rostermodel.SubscriptionFrom || ri.Subscription == rostermodel.SubscriptionBoth)
	return allowed, nil
}

func (ac *accessChecker) checkRosterAccess(host, j string) (bool, error) {
	userJID, _ := jid.NewWithString(host, true)
	contactJID, _ := jid.NewWithString(j, true)

	ri, err := storage.FetchRosterItem(userJID.Node(), contactJID.ToBareJID().String())
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

func (ac *accessChecker) checkWhitelistAccess(j string) bool {
	for _, aff := range ac.affiliations {
		if aff.JID != j {
			continue
		}
		switch aff.Affiliation {
		case pubsubmodel.Owner, pubsubmodel.Member:
			return true
		}
	}
	return false
}
