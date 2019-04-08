package offline

import "fmt"

const (
	httpGatewayType = "http"
)

// Config represents Offline Storage module configuration.
type Config struct {
	QueueSize int
	Gateway   gateway
}

type configProxy struct {
	QueueSize int `yaml:"queue_size"`
	Gateway   *struct {
		Type string `yaml:"type"`
		Pass string `yaml:"pass"`
	} `yaml:"gateway"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (cfg *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := configProxy{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	cfg.QueueSize = p.QueueSize
	if p.Gateway != nil {
		switch p.Gateway.Type {
		case httpGatewayType:
			cfg.Gateway = newHTTPGateway(p.Gateway.Pass)
		default:
			return fmt.Errorf("unrecognized offline gateway type: %s", p.Gateway.Type)
		}
	}
	return nil
}
