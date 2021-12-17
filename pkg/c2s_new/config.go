package c2s_new

import "time"

// Config defines C2S subsystem configuration.
type Config struct {
	Listeners []ListenerConfig `fig:"listeners"`
}

// ListenerConfig contains a C2S listener configuration.
type ListenerConfig struct {
	// BindAddr defines listener incoming connections address.
	BindAddr string `fig:"bind_addr"`

	// Port defines listener incoming connections port.
	Port int `fig:"port" default:"5222"`

	// Transport specifies the type of transport used for incoming connections.
	Transport string `fig:"transport" default:"socket"`

	// DirectTLS, if true, tls.Listen will be used as network listener.
	DirectTLS bool `fig:"direct_tls"`

	// SASL contains authentication related configuration.
	SASL struct {
		// Mechanisms contains enabled SASL mechanisms.
		Mechanisms []string `fig:"mechanisms" default:"[scram_sha_1, scram_sha_256, scram_sha_512, scram_sha3_512]"`

		// External contains external authenticator configuration.
		External struct {
			Address  string `fig:"address"`
			IsSecure bool   `fig:"is_secure"`
		} `fig:"external"`
	} `fig:"sasl"`

	// CompressionLevel is the compression level that may be applied to the stream.
	// Valid values are 'default', 'best', 'speed' and 'no_compression'.
	CompressionLevel string `fig:"compression_level" default:"default"`

	// ResourceConflict defines the which rule should be applied in a resource conflict is detected.
	// Valid values are `override`, `disallow` and `terminate_old`.
	ResourceConflict string `fig:"resource_conflict" default:"terminate_old"`

	// MaxStanzaSize is the maximum size a listener incoming stanza may have.
	MaxStanzaSize int `fig:"max_stanza_size" default:"32768"`

	// ConnectTimeout defines connection timeout.
	ConnectTimeout time.Duration `fig:"conn_timeout" default:"3s"`

	// AuthenticateTimeout defines authentication timeout.
	AuthenticateTimeout time.Duration `fig:"auth_timeout" default:"10s"`

	// KeepAliveTimeout defines the maximum amount of time that an inactive connection
	// would be considered alive.
	KeepAliveTimeout time.Duration `fig:"keep_alive_timeout" default:"10m"`

	// RequestTimeout defines C2S stream request timeout.
	RequestTimeout time.Duration `fig:"req_timeout" default:"15s"`
}
