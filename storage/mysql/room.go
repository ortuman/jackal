/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/util/pool"
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
	columns := []string{"name"}
	values := []interface{}{room.Name}

	q := sq.Insert("rooms").
		Columns(columns...).
		Values(values...)

	_, err := q.RunWith(r.db).ExecContext(ctx)
	return err
}

func (r *mySQLRoom) FetchRoom(ctx context.Context, roomName string) (*mucmodel.Room, error) {
	q := sq.Select("name").
		From("rooms").
		Where(sq.Eq{"name": roomName})

	var room mucmodel.Room

	err := q.RunWith(r.db).
		QueryRowContext(ctx).
		Scan(&room.Name)
	switch err {
	case nil:
		return &room, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (r *mySQLRoom) DeleteRoom(ctx context.Context, roomName string) error {
	return r.inTransaction(ctx, func(tx *sql.Tx) error {
		var err error
		_, err = sq.Delete("rooms").Where(sq.Eq{"name": roomName}).RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		return nil
	})
}

func (r *mySQLRoom) RoomExists(ctx context.Context, roomName string) (bool, error) {
	q := sq.Select("COUNT(*)").
		From("rooms").
		Where(sq.Eq{"roomName": roomName})

	var count int
	err := q.RunWith(r.db).QueryRowContext(ctx).Scan(&count)
	switch err {
	case nil:
		return count > 0, nil
	default:
		return false, err
	}
}
