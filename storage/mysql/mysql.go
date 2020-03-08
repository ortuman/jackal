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

	_ "github.com/go-sql-driver/mysql" // SQL driver
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage/repository"
)

type mySQLContainer struct {
	user      *mySQLUser
	roster    *mySQLRoster
	presences *mySQLPresences
	vCard     *mySQLVCard
	priv      *mySQLPrivate
	blockList *mySQLBlockList
	pubSub    *mySQLPubSub
	offline   *mySQLOffline

	h      *sql.DB
	doneCh chan chan bool
}

// New initializes MySQL storage and returns associated container.
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
	c.roster = newRoster(c.h)
	c.presences = newPresences(c.h)
	c.vCard = newVCard(c.h)
	c.priv = newPrivate(c.h)
	c.blockList = newBlockList(c.h)
	c.pubSub = newPubSub(c.h)
	c.offline = newOffline(c.h)

	return c, nil
}

func (c *mySQLContainer) User() repository.User           { return c.user }
func (c *mySQLContainer) Roster() repository.Roster       { return c.roster }
func (c *mySQLContainer) Presences() repository.Presences { return c.presences }
func (c *mySQLContainer) VCard() repository.VCard         { return c.vCard }
func (c *mySQLContainer) Private() repository.Private     { return c.priv }
func (c *mySQLContainer) BlockList() repository.BlockList { return c.blockList }
func (c *mySQLContainer) PubSub() repository.PubSub       { return c.pubSub }
func (c *mySQLContainer) Offline() repository.Offline     { return c.offline }

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
