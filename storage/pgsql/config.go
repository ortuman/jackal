package pgsql

// DefaultPoolSize defines the default size of the database connection pool
const DefaultPoolSize = 16

// Config represents PostgreSQL storage configuration.
type Config struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	PoolSize int    `yaml:"pool_size"`
}

// UnmarshalYAML satisfies Unmarshaler interface
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawConfig Config

	parsed := rawConfig{PoolSize: DefaultPoolSize}

	if err := unmarshal(&parsed); err != nil {
		return err
	}

	*c = Config(parsed)

	return nil
}
