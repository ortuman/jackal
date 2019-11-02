package storage

type capabilitiesStorage interface {
	InsertCapabilities(node, ver string, features []string) error

	HasCapabilities(node, ver string) (bool, error)
	FetchCapabilities(node, ver string) ([]string, error)
}

func InsertCapabilities(node, ver string, features []string) error {
	return inst.InsertCapabilities(node, ver, features)
}

func HasCapabilities(node, ver string) (bool, error) {
	return inst.HasCapabilities(node, ver)
}

func FetchCapabilities(node, ver string) ([]string, error) {
	return inst.FetchCapabilities(node, ver)
}
