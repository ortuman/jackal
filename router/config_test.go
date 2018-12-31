/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestConfig_BadFormat(t *testing.T) {
	s := Config{}

	err := yaml.Unmarshal([]byte("{["), &s)
	require.NotNil(t, err)

	err = yaml.Unmarshal([]byte("{}"), &s)
	require.NotNil(t, err)

	cfg := `
  hosts:
    - name: jackal.im
      tls:
        privkey_path: "key.pem"
        cert_path: "cert.pem"
`
	err = yaml.Unmarshal([]byte(cfg), &s)
	require.NotNil(t, err)
}

func TestConfig_Valid(t *testing.T) {
	defer os.RemoveAll("./.cert")

	s := Config{}

	cfg := `
  hosts:
    - name: localhost
      tls:
        privkey_path: ""
        cert_path: ""
`
	err := yaml.Unmarshal([]byte(cfg), &s)
	require.Nil(t, err)
}
