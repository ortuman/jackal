/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"context"
	"database/sql"

	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/util/pool"
	"github.com/ortuman/jackal/xmpp/jid"
)

type mySQLRoom struct {
	*mySQLStorage
	pool *pool.BufferPool
}

func newRoom(db *sql.DB) *mySQLRoom {
	return &mySQLRoom{
		mySQLStorage: newStorage(db),
		pool:         pool.NewBufferPool(),
	}
}

func (r *mySQLRoom) UpsertRoom(ctx context.Context, room *mucmodel.Room) error {
	return nil
}

func (r *mySQLRoom) FetchRoom(ctx context.Context, roomJID *jid.JID) (*mucmodel.Room, error) {
	return nil, nil
}

func (r *mySQLRoom) DeleteRoom(ctx context.Context, roomJID *jid.JID) error {
	return nil
}

func (r *mySQLRoom) RoomExists(ctx context.Context, roomJID *jid.JID) (bool, error) {
	return false, nil
}
