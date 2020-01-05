/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"bytes"
	"context"
	"encoding/gob"

	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/model/serializer"
)

type Roster struct {
	*memoryStorage
}

func NewRoster() *Roster {
	return &Roster{memoryStorage: newStorage()}
}

func (m *Roster) UpsertRosterItem(_ context.Context, ri *rostermodel.Item) (rostermodel.Version, error) {
	var rv rostermodel.Version
	err := m.inWriteLock(func() error {
		ris, fnErr := m.fetchRosterItems(ri.Username)
		if fnErr != nil {
			return fnErr
		}
		if ris != nil {
			for i, r := range ris {
				if r.JID == ri.JID {
					ris[i] = *ri
					goto done
				}
			}
			ris = append(ris, *ri)
		} else {
			ris = []rostermodel.Item{*ri}
		}

	done:
		if fnErr := m.upsertRosterGroups(ri.Username, ris); fnErr != nil {
			return fnErr
		}
		rv, fnErr = m.fetchRosterVersion(ri.Username)
		if fnErr != nil {
			return fnErr
		}
		rv.Ver++
		if err := m.upsertRosterVersion(rv, ri.Username); err != nil {
			return err
		}
		ris[len(ris)-1].Ver = rv.Ver
		return m.upsertRosterItems(ris, ri.Username)
	})
	return rv, err
}

func (m *Roster) DeleteRosterItem(_ context.Context, user, contact string) (rostermodel.Version, error) {
	var rv rostermodel.Version
	if err := m.inWriteLock(func() error {
		ris, fnErr := m.fetchRosterItems(user)
		if fnErr != nil {
			return fnErr
		}
		for i, ri := range ris {
			if ri.JID == contact {
				ris = append(ris[:i], ris[i+1:]...)
				if err := m.upsertRosterItems(ris, user); err != nil {
					return err
				}
				goto done
			}
		}
	done:
		if fnErr := m.upsertRosterGroups(user, ris); fnErr != nil {
			return fnErr
		}
		rv, fnErr = m.fetchRosterVersion(user)
		if fnErr != nil {
			return fnErr
		}
		rv.Ver++
		rv.DeletionVer = rv.Ver
		return m.upsertRosterVersion(rv, user)
	}); err != nil {
		return rostermodel.Version{}, err
	}
	return rv, nil
}

func (m *Roster) FetchRosterItems(_ context.Context, user string) ([]rostermodel.Item, rostermodel.Version, error) {
	var ris []rostermodel.Item
	var rv rostermodel.Version

	if err := m.inReadLock(func() error {
		var fnErr error
		ris, fnErr = m.fetchRosterItems(user)
		if fnErr != nil {
			return fnErr
		}
		rv, fnErr = m.fetchRosterVersion(user)
		return fnErr
	}); err != nil {
		return nil, rostermodel.Version{}, err
	}
	return ris, rv, nil
}

func (m *Roster) FetchRosterItemsInGroups(_ context.Context, username string, groups []string) ([]rostermodel.Item, rostermodel.Version, error) {
	var ris []rostermodel.Item
	var rv rostermodel.Version

	groupSet := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		groupSet[group] = struct{}{}
	}
	if err := m.inReadLock(func() error {
		fnRis, fnErr := m.fetchRosterItems(username)
		if fnErr != nil {
			return fnErr
		}
		for _, ri := range fnRis {
			for _, riGroup := range ri.Groups {
				if _, ok := groupSet[riGroup]; ok {
					ris = append(ris, ri)
					break
				}
			}
		}
		rv, fnErr = m.fetchRosterVersion(username)
		return fnErr
	}); err != nil {
		return nil, rostermodel.Version{}, err
	}
	return ris, rv, nil
}

func (m *Roster) FetchRosterItem(_ context.Context, user, contact string) (*rostermodel.Item, error) {
	var ret *rostermodel.Item
	err := m.inReadLock(func() error {
		ris, fnErr := m.fetchRosterItems(user)
		if fnErr != nil {
			return fnErr
		}
		for _, ri := range ris {
			if ri.JID == contact {
				ret = &ri
				return nil
			}
		}
		return nil
	})
	return ret, err
}

func (m *Roster) UpsertRosterNotification(_ context.Context, rn *rostermodel.Notification) error {
	return m.inWriteLock(func() error {
		rns, fnErr := m.fetchRosterNotifications(rn.Contact)
		if fnErr != nil {
			return fnErr
		}
		if rns != nil {
			for i, r := range rns {
				if r.JID == rn.JID {
					rns[i] = *rn
					goto done
				}
			}
			rns = append(rns, *rn)
		} else {
			rns = []rostermodel.Notification{*rn}
		}
	done:
		return m.upsertRosterNotifications(rns, rn.Contact)
	})
}

func (m *Roster) DeleteRosterNotification(_ context.Context, contact, jid string) error {
	return m.inWriteLock(func() error {
		rns, fnErr := m.fetchRosterNotifications(contact)
		if fnErr != nil {
			return fnErr
		}
		for i, rn := range rns {
			if rn.JID == jid {
				rns = append(rns[:i], rns[i+1:]...)
				return m.upsertRosterNotifications(rns, contact)
			}
		}
		return nil
	})
}

func (m *Roster) FetchRosterNotification(_ context.Context, contact string, jid string) (*rostermodel.Notification, error) {
	var ret *rostermodel.Notification
	err := m.inReadLock(func() error {
		rns, fnErr := m.fetchRosterNotifications(contact)
		if fnErr != nil {
			return fnErr
		}
		for _, rn := range rns {
			if rn.JID == jid {
				ret = &rn
				break
			}
		}
		return nil
	})
	return ret, err
}

func (m *Roster) FetchRosterNotifications(_ context.Context, contact string) ([]rostermodel.Notification, error) {
	var rns []rostermodel.Notification
	if err := m.inReadLock(func() error {
		var fnErr error
		rns, fnErr = m.fetchRosterNotifications(contact)
		return fnErr
	}); err != nil {
		return nil, err
	}
	return rns, nil
}

func (m *Roster) FetchRosterGroups(_ context.Context, username string) ([]string, error) {
	var groups []string
	if err := m.inReadLock(func() error {
		var fnErr error
		groups, fnErr = m.fetchRosterGroups(username)
		return fnErr
	}); err != nil {
		return nil, err
	}
	return groups, nil
}

func (m *Roster) upsertRosterItems(ris []rostermodel.Item, user string) error {
	b, err := serializer.SerializeSlice(&ris)
	if err != nil {
		return err
	}
	m.b[rosterItemsKey(user)] = b
	return nil
}

func (m *Roster) fetchRosterItems(user string) ([]rostermodel.Item, error) {
	b := m.b[rosterItemsKey(user)]
	if b == nil {
		return nil, nil
	}
	var ris []rostermodel.Item
	if err := serializer.DeserializeSlice(b, &ris); err != nil {
		return nil, err
	}
	return ris, nil
}

func (m *Roster) upsertRosterVersion(rv rostermodel.Version, user string) error {
	b, err := serializer.Serialize(&rv)
	if err != nil {
		return err
	}
	m.b[rosterVersionKey(user)] = b
	return nil
}

func (m *Roster) fetchRosterVersion(user string) (rostermodel.Version, error) {
	b := m.b[rosterVersionKey(user)]
	if b == nil {
		return rostermodel.Version{}, nil
	}
	var rv rostermodel.Version
	if err := serializer.Deserialize(b, &rv); err != nil {
		return rostermodel.Version{}, err
	}
	return rv, nil
}

func (m *Roster) upsertRosterNotifications(rns []rostermodel.Notification, contact string) error {
	b, err := serializer.SerializeSlice(&rns)
	if err != nil {
		return err
	}
	m.b[rosterNotificationsKey(contact)] = b
	return nil
}

func (m *Roster) fetchRosterNotifications(contact string) ([]rostermodel.Notification, error) {
	b := m.b[rosterNotificationsKey(contact)]
	if b == nil {
		return nil, nil
	}
	var rns []rostermodel.Notification
	if err := serializer.DeserializeSlice(b, &rns); err != nil {
		return nil, err
	}
	return rns, nil
}

func (m *Roster) upsertRosterGroups(user string, ris []rostermodel.Item) error {
	var groupsSet = make(map[string]struct{})
	// remove duplicates
	for _, ri := range ris {
		for _, group := range ri.Groups {
			groupsSet[group] = struct{}{}
		}
	}
	var groups []string
	for group := range groupsSet {
		groups = append(groups, group)
	}
	// encode groups
	buf := bytes.NewBuffer(nil)

	enc := gob.NewEncoder(buf)
	if err := enc.Encode(len(groups)); err != nil {
		return err
	}
	for _, group := range groups {
		if err := enc.Encode(group); err != nil {
			return err
		}
	}
	m.b[rosterGroupsKey(user)] = buf.Bytes()
	return nil
}

func (m *Roster) fetchRosterGroups(user string) ([]string, error) {
	var ln int
	var groups []string

	b := m.b[rosterGroupsKey(user)]
	if b == nil {
		return nil, nil
	}
	// decode groups
	dec := gob.NewDecoder(bytes.NewReader(b))
	if err := dec.Decode(&ln); err != nil {
		return nil, err
	}
	for i := 0; i < ln; i++ {
		var group string
		if err := dec.Decode(&group); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func rosterItemsKey(user string) string {
	return "rosterItems:" + user
}

func rosterVersionKey(username string) string {
	return "rosterVersions:" + username
}

func rosterNotificationsKey(contact string) string {
	return "rosterNotifications:" + contact
}

func rosterGroupsKey(username string) string {
	return "rosterGroups:" + username
}
