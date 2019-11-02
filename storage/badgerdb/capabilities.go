package badgerdb

func (b *Storage) InsertCapabilities(node, ver string, features []string) error {
	// TODO(ortuman): Implement me!
	return nil
}

func (b *Storage) HasCapabilities(node, ver string) (bool, error) {
	// TODO(ortuman): Implement me!
	return false, nil
}

func (b *Storage) FetchCapabilities(node, ver string) ([]string, error) {
	// TODO(ortuman): Implement me!
	return nil, nil
}
