/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"strings"

	"github.com/ortuman/jackal/storage/model"
)

type rowScanner interface {
	Scan(...interface{}) error
}

type rowsScanner interface {
	rowScanner
	Next() bool
}

func scanRosterItemEntity(ri *model.RosterItem, scanner rowScanner) error {
	var groups string
	if err := scanner.Scan(&ri.User, &ri.Contact, &ri.Name, &ri.Subscription, &groups, &ri.Ask); err != nil {
		return err
	}
	ri.Groups = strings.Split(groups, ";")
	return nil
}

func scanRosterItemEntities(scanner rowsScanner) ([]model.RosterItem, error) {
	var ret []model.RosterItem
	for scanner.Next() {
		var ri model.RosterItem
		if err := scanRosterItemEntity(&ri, scanner); err != nil {
			return nil, err
		}
		ret = append(ret, ri)
	}
	return ret, nil
}
