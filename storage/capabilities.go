package storage

import "github.com/ortuman/jackal/model"

type capabilitiesStorage interface {
	InsertCapabilities(caps *model.Capabilities) error

	HasCapabilities(node, ver string) (bool, error)
	FetchCapabilities(node, ver string) (*model.Capabilities, error)
}

func InsertCapabilities(caps *model.Capabilities) error {
	return inst.InsertCapabilities(caps)
}

func HasCapabilities(node, ver string) (bool, error) {
	return inst.HasCapabilities(node, ver)
}

func FetchCapabilities(node, ver string) (*model.Capabilities, error) {
	return inst.FetchCapabilities(node, ver)
}
