package storage

import (
	"context"

	"github.com/ortuman/jackal/model"
)

type capabilitiesStorage interface {
	InsertCapabilities(ctx context.Context, caps *model.Capabilities) error
	FetchCapabilities(ctx context.Context, node, ver string) (*model.Capabilities, error)
}

func InsertCapabilities(ctx context.Context, caps *model.Capabilities) error {
	return inst.InsertCapabilities(ctx, caps)
}

func FetchCapabilities(ctx context.Context, node, ver string) (*model.Capabilities, error) {
	return inst.FetchCapabilities(ctx, node, ver)
}
