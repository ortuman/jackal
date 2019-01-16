package pgsql

// defaultPoolSize defines the default size of the database connection pool
const defaultPoolSize = 16
const defaultSSLMode = "disable"

// Config represents PostgreSQL storage configuration.
type Config struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	PoolSize int    `yaml:"pool_size"`
	SSLMode  string `yaml:"ssl_mode"`
}

// UnmarshalYAML satisfies Unmarshaler interface
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawConfig Config

	parsed := rawConfig{
		PoolSize: defaultPoolSize,
		SSLMode:  defaultSSLMode,
	}

	if err := unmarshal(&parsed); err != nil {
		return err
	}

	*c = Config(parsed)

	return nil
}
