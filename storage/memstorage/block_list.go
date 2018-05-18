/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import "github.com/ortuman/jackal/storage/model"

func (m *Storage) InsertOrUpdateBlockListItems(items []model.BlockListItem) error {
	return m.inWriteLock(func() error {
		for _, item := range items {
			bl := m.blockListItems[item.Username]
			if bl != nil {
				for _, blItem := range bl {
					if blItem.JID == item.JID {
						goto done
					}
				}
				m.blockListItems[item.Username] = append(bl, item)
			} else {
				m.blockListItems[item.Username] = []model.BlockListItem{item}
			}
		done:
		}
		return nil
	})
}

func (m *Storage) DeleteBlockListItems(items []model.BlockListItem) error {
	return m.inWriteLock(func() error {
		for _, itm := range items {
			bl := m.blockListItems[itm.Username]
			for i, blItem := range bl {
				if blItem.JID == itm.JID {
					m.blockListItems[itm.Username] = append(bl[:i], bl[i+1:]...)
					break
				}
			}
		}
		return nil
	})
}

func (m *Storage) FetchBlockListItems(username string) ([]model.BlockListItem, error) {
	var ret []model.BlockListItem
	err := m.inReadLock(func() error {
		ret = m.blockListItems[username]
		return nil
	})
	return ret, err
}
