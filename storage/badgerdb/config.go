package badgerdb

// Config represents BadgerDB storage configuration.
type Config struct {
	DataDir string `yaml:"data_dir"`
}

// DefaultDataDir is the default directory for BadgerDB storage
const DefaultDataDir = "./data"

// UnmarshalYAML satisfies Unmarshaler interface
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawConfig Config

	parsed := rawConfig{DataDir: DefaultDataDir}

	if err := unmarshal(&parsed); err != nil {
		return err
	}

	*c = Config(parsed)

	return nil
}
