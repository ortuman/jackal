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
	user    *User
	vCard   *VCard
	private *Private
}

func New() (repository.Container, error) {
	var c memoryContainer

	c.user = NewUser()
	c.vCard = NewVCard()
	c.private = NewPrivate()
	return &c, nil
}

func (c *memoryContainer) User() repository.User       { return c.user }
func (c *memoryContainer) VCard() repository.VCard     { return c.vCard }
func (c *memoryContainer) Private() repository.Private { return c.private }

func (c *memoryContainer) Close(_ context.Context) error { return nil }

func (c *memoryContainer) IsClusterCompatible() bool { return false }
