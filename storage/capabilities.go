package storage

import "github.com/ortuman/jackal/model"

type capabilitiesStorage interface {
	InsertCapabilities(caps *model.Capabilities) error
	FetchCapabilities(node, ver string) (*model.Capabilities, error)
}

func InsertCapabilities(caps *model.Capabilities) error {
	return inst.InsertCapabilities(caps)
}

func FetchCapabilities(node, ver string) (*model.Capabilities, error) {
	return inst.FetchCapabilities(node, ver)
}
