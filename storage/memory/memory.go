/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"

	"github.com/ortuman/jackal/storage/repository"
)

type memoryContainer struct {
	user      *User
	roster    *Roster
	vCard     *VCard
	caps      *Capabilities
	priv      *Private
	blockList *BlockList
	pubSub    *PubSub
	offline   *Offline
}

func New() (repository.Container, error) {
	var c memoryContainer

	c.user = NewUser()
	c.roster = NewRoster()
	c.caps = NewCapabilities()
	c.vCard = NewVCard()
	c.priv = NewPrivate()
	c.blockList = NewBlockList()
	c.pubSub = NewPubSub()
	c.offline = NewOffline()

	return &c, nil
}

func (c *memoryContainer) User() repository.User                 { return c.user }
func (c *memoryContainer) Roster() repository.Roster             { return c.roster }
func (c *memoryContainer) Capabilities() repository.Capabilities { return c.caps }
func (c *memoryContainer) VCard() repository.VCard               { return c.vCard }
func (c *memoryContainer) Private() repository.Private           { return c.priv }
func (c *memoryContainer) BlockList() repository.BlockList       { return c.blockList }
func (c *memoryContainer) PubSub() repository.PubSub             { return c.pubSub }
func (c *memoryContainer) Offline() repository.Offline           { return c.offline }

func (c *memoryContainer) Close(_ context.Context) error { return nil }

func (c *memoryContainer) IsClusterCompatible() bool { return false }
