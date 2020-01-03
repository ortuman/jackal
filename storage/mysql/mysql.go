/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage/repository"
)

type mySQLContainer struct {
	user  *mySQLUser
	caps  *mySQLCapabilities
	vCard *mySQLVCard
	priv  *mySQLPrivate

	h      *sql.DB
	doneCh chan chan bool
}

func New(cfg *Config) (repository.Container, error) {
	var err error
	c := &mySQLContainer{doneCh: make(chan chan bool, 1)}
	host := cfg.Host
	usr := cfg.User
	pass := cfg.Password
	db := cfg.Database
	poolSize := cfg.PoolSize

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", usr, pass, host, db)
	c.h, err = sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	c.h.SetMaxOpenConns(poolSize) // set max opened connection count

	if err := c.h.Ping(); err != nil {
		return nil, err
	}
	go c.loop()

	c.user = newUser(c.h)
	c.caps = newCapabilities(c.h)
	c.vCard = newVCard(c.h)
	c.priv = newPrivate(c.h)
	return c, nil
}

func (c *mySQLContainer) User() repository.User                 { return c.user }
func (c *mySQLContainer) Capabilities() repository.Capabilities { return c.caps }
func (c *mySQLContainer) VCard() repository.VCard               { return c.vCard }
func (c *mySQLContainer) Private() repository.Private           { return c.priv }

func (c *mySQLContainer) Close(ctx context.Context) error {
	ch := make(chan bool)
	c.doneCh <- ch
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *mySQLContainer) IsClusterCompatible() bool { return true }

func (c *mySQLContainer) loop() {
	tc := time.NewTicker(time.Second * 15)
	defer tc.Stop()

	for {
		select {
		case <-tc.C:
			if err := c.h.Ping(); err != nil {
				log.Error(err)
			}
		case ch := <-c.doneCh:
			if err := c.h.Close(); err != nil {
				log.Error(err)
			}
			close(ch)
			return
		}
	}
}
