/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage/repository"
)

// pingInterval defines how often to check the connection
var pingInterval = 15 * time.Second

// pingTimeout defines how long to wait for pong from server
var pingTimeout = 10 * time.Second

type pgSQLContainer struct {
	user  *pgSQLUser
	caps  *pgSQLCapabilities
	vCard *pgSQLVCard
	priv  *pgSQLPrivate

	h          *sql.DB
	cancelPing context.CancelFunc
	doneCh     chan chan bool
}

func New(cfg *Config) (repository.Container, error) {
	c := &pgSQLContainer{doneCh: make(chan chan bool, 1)}

	var err error

	sq.StatementBuilder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s", cfg.User, cfg.Password, cfg.Host, cfg.Database, cfg.SSLMode)

	c.h, err = sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	c.h.SetMaxOpenConns(cfg.PoolSize) // set max opened connection count

	if err := c.ping(context.Background()); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelPing = cancel
	go c.loop(ctx)

	c.user = newUser(c.h)
	c.caps = newCapabilities(c.h)
	c.vCard = newVCard(c.h)
	c.priv = newPrivate(c.h)
	return c, nil
}

func (c *pgSQLContainer) User() repository.User                 { return c.user }
func (c *pgSQLContainer) Capabilities() repository.Capabilities { return c.caps }
func (c *pgSQLContainer) VCard() repository.VCard               { return c.vCard }
func (c *pgSQLContainer) Private() repository.Private           { return c.priv }

func (c *pgSQLContainer) Close(ctx context.Context) error {
	ch := make(chan bool)
	c.doneCh <- ch
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *pgSQLContainer) IsClusterCompatible() bool { return true }

func (c *pgSQLContainer) loop(ctx context.Context) {
	tick := time.NewTicker(pingInterval)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			if err := c.ping(ctx); err != nil {
				log.Error(err)
			}

		case ch := <-c.doneCh:
			if err := c.h.Close(); err != nil {
				log.Error(err)
			}
			close(ch)
			return

		case <-ctx.Done():
			return
		}
	}
}

// ping sends a ping request to the server and outputs any error to log
func (c *pgSQLContainer) ping(ctx context.Context) error {
	pingCtx, cancel := context.WithDeadline(ctx, time.Now().Add(pingTimeout))
	defer cancel()

	return c.h.PingContext(pingCtx)
}
