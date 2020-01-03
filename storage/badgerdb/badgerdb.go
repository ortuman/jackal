/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"context"
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/storage/repository"
)

type badgerDBContainer struct {
	user    *badgerDBUser
	vCard   *badgerDBVCard
	private *badgerDBPrivate

	db *badger.DB
}

func New(cfg *Config) (repository.Container, error) {
	var c badgerDBContainer

	if err := os.MkdirAll(filepath.Dir(cfg.DataDir), os.ModePerm); err != nil {
		return nil, err
	}
	opts := badger.DefaultOptions(cfg.DataDir)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	c.db = db

	c.user = newUser(c.db)
	c.vCard = newVCard(c.db)
	c.private = newPrivate(c.db)

	return &c, nil
}

func (c *badgerDBContainer) User() repository.User       { return c.user }
func (c *badgerDBContainer) VCard() repository.VCard     { return c.vCard }
func (c *badgerDBContainer) Private() repository.Private { return c.private }

func (c *badgerDBContainer) Close(_ context.Context) error { return c.db.Close() }

func (c *badgerDBContainer) IsClusterCompatible() bool { return false }
