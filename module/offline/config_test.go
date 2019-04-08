package offline

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestOfflineConfig(t *testing.T) {
	badCfg := `enabled [roster]`
	cfg := &Config{}
	err := yaml.Unmarshal([]byte(badCfg), &cfg)
	require.NotNil(t, err)

	goodCfg := `queue_size: 100`
	cfg = &Config{}
	err = yaml.Unmarshal([]byte(goodCfg), &cfg)
	require.Nil(t, err)

	wrongGatewayTypeCfg := `
queue_size: 100
gateway:
	type: foo
	url: http://127.0.0.1:6666
`
	cfg = &Config{}
	err = yaml.Unmarshal([]byte(wrongGatewayTypeCfg), &cfg)
	require.NotNil(t, err)

	goodGatewayTypeCfg := `
queue_size: 100
gateway:
	type: http
	url: http://127.0.0.1:6666
`
	cfg = &Config{}
	err = yaml.Unmarshal([]byte(goodGatewayTypeCfg), &cfg)
	require.NotNil(t, err)
}
