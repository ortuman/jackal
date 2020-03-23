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
	presences *Presences
	vCard     *VCard
	priv      *Private
	blockList *BlockList
	pubSub    *PubSub
	offline   *Offline
	room      *Room
	occ       *Occupant
}

// New initializes in-memory storage and returns associated container.
func New() (repository.Container, error) {
	var c memoryContainer

	c.user = NewUser()
	c.roster = NewRoster()
	c.presences = NewPresences()
	c.vCard = NewVCard()
	c.priv = NewPrivate()
	c.blockList = NewBlockList()
	c.pubSub = NewPubSub()
	c.offline = NewOffline()
	c.room = NewRoom()
	c.occ = NewOccupant()

	return &c, nil
}

func (c *memoryContainer) User() repository.User           { return c.user }
func (c *memoryContainer) Roster() repository.Roster       { return c.roster }
func (c *memoryContainer) Presences() repository.Presences { return c.presences }
func (c *memoryContainer) VCard() repository.VCard         { return c.vCard }
func (c *memoryContainer) Private() repository.Private     { return c.priv }
func (c *memoryContainer) BlockList() repository.BlockList { return c.blockList }
func (c *memoryContainer) PubSub() repository.PubSub       { return c.pubSub }
func (c *memoryContainer) Offline() repository.Offline     { return c.offline }

func (c *memoryContainer) Close(_ context.Context) error { return nil }

func (c *memoryContainer) IsClusterCompatible() bool     { return false }
func (c *memoryContainer) Room() repository.Room         { return c.room }
func (c *memoryContainer) Occupant() repository.Occupant { return c.occ }
