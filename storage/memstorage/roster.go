/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import "github.com/ortuman/jackal/storage/model"

// InsertOrUpdateRosterItem inserts a new roster item entity into storage,
// or updates it in case it's been previously inserted.
func (m *Storage) InsertOrUpdateRosterItem(ri *model.RosterItem) (model.RosterVersion, error) {
	var v model.RosterVersion
	err := m.inWriteLock(func() error {
		ris := m.rosterItems[ri.Username]
		if ris != nil {
			for i, r := range ris {
				if r.JID == ri.JID {
					ris[i] = *ri
					goto done
				}
			}
			ris = append(ris, *ri)
		} else {
			ris = []model.RosterItem{*ri}
		}

	done:
		ver := m.rosterVersions[ri.Username]
		ver.Ver++
		m.rosterVersions[ri.Username] = ver
		ris[len(ris)-1].Ver = ver.Ver
		m.rosterItems[ri.Username] = ris
		return nil
	})
	return v, err
}

// DeleteRosterItem deletes a roster item entity from storage.
func (m *Storage) DeleteRosterItem(user, contact string) (model.RosterVersion, error) {
	var v model.RosterVersion
	err := m.inWriteLock(func() error {
		ris := m.rosterItems[user]
		for i, ri := range ris {
			if ri.JID == contact {
				m.rosterItems[user] = append(ris[:i], ris[i+1:]...)
				goto done
			}
		}
	done:
		v = m.rosterVersions[user]
		v.Ver++
		v.DeletionVer = v.Ver
		m.rosterVersions[user] = v
		return nil
	})
	return v, err
}

// FetchRosterItems retrieves from storage all roster item entities
// associated to a given user.
func (m *Storage) FetchRosterItems(user string) ([]model.RosterItem, model.RosterVersion, error) {
	var ris []model.RosterItem
	var v model.RosterVersion
	err := m.inReadLock(func() error {
		ris = m.rosterItems[user]
		v = m.rosterVersions[user]
		return nil
	})
	return ris, v, err
}

// FetchRosterItem retrieves from storage a roster item entity.
func (m *Storage) FetchRosterItem(user, contact string) (*model.RosterItem, error) {
	var ret *model.RosterItem
	err := m.inReadLock(func() error {
		ris := m.rosterItems[user]
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

// InsertOrUpdateRosterNotification inserts a new roster notification entity
// into storage, or updates it in case it's been previously inserted.
func (m *Storage) InsertOrUpdateRosterNotification(rn *model.RosterNotification) error {
	return m.inWriteLock(func() error {
		rns := m.rosterNotifications[rn.Contact]
		if rns != nil {
			for i, r := range rns {
				if r.JID == rn.JID {
					rns[i] = *rn
					goto done
				}
			}
			rns = append(rns, *rn)
		} else {
			rns = []model.RosterNotification{*rn}
		}
	done:
		m.rosterNotifications[rn.Contact] = rns
		return nil
	})
}

// DeleteRosterNotification deletes a roster notification entity from storage.
func (m *Storage) DeleteRosterNotification(contact, jid string) error {
	return m.inWriteLock(func() error {
		rns := m.rosterNotifications[contact]
		for i, rn := range rns {
			if rn.JID == jid {
				m.rosterNotifications[contact] = append(rns[:i], rns[i+1:]...)
				return nil
			}
		}
		return nil
	})
}

// FetchRosterNotifications retrieves from storage all roster notifications
// associated to a given user.
func (m *Storage) FetchRosterNotifications(contact string) ([]model.RosterNotification, error) {
	var ret []model.RosterNotification
	err := m.inReadLock(func() error {
		ret = m.rosterNotifications[contact]
		return nil
	})
	return ret, err
}
