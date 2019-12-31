/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memory

import (
	"context"

	"github.com/ortuman/jackal/storage/repository"
)

type memoryContainer struct {
	user *user
}

func New() (repository.Container, error) {
	var c memoryContainer

	c.user = newUser()
	return &c, nil
}

func (c *memoryContainer) User() repository.User { return c.user }

func (c *memoryContainer) Close(_ context.Context) error { return nil }

func (c *memoryContainer) IsClusterCompatible() bool { return false }
