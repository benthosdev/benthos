package output

import (
	"github.com/benthosdev/benthos/v4/internal/batch/policy/batchconfig"
	"github.com/benthosdev/benthos/v4/internal/old/util/retries"
	btls "github.com/benthosdev/benthos/v4/internal/tls"
)

// PasswordAuthenticator contains the fields that will be used to authenticate with
// the Cassandra cluster.
type PasswordAuthenticator struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

// CassandraConfig contains configuration fields for the Cassandra output type.
type CassandraConfig struct {
	Addresses                []string              `json:"addresses" yaml:"addresses"`
	TLS                      btls.Config           `json:"tls" yaml:"tls"`
	PasswordAuthenticator    PasswordAuthenticator `json:"password_authenticator" yaml:"password_authenticator"`
	DisableInitialHostLookup bool                  `json:"disable_initial_host_lookup" yaml:"disable_initial_host_lookup"`
	Query                    string                `json:"query" yaml:"query"`
	ArgsMapping              string                `json:"args_mapping" yaml:"args_mapping"`
	Consistency              string                `json:"consistency" yaml:"consistency"`
	Timeout                  string                `json:"timeout" yaml:"timeout"`
	ConnectTimeout           string                `json:"connect_timeout" yaml:"connect_timeout"`
	LoggedBatch              bool                  `json:"logged_batch" yaml:"logged_batch"`
	UseTokenAwareHostPolicy  bool                  `json:"use_token_aware_host_policy" yaml:"use_token_aware_host_policy"`
	ShuffleReplicas          bool                  `json:"shuffle_replicas" yaml:"shuffle_replicas"`
	UseCompressor            bool                  `json:"use_compressor" yaml:"use_compressor"`
	// TODO: V4 Remove this and replace with explicit values.
	retries.Config `json:",inline" yaml:",inline"`
	MaxInFlight    int                `json:"max_in_flight" yaml:"max_in_flight"`
	Batching       batchconfig.Config `json:"batching" yaml:"batching"`
}

// NewCassandraConfig creates a new CassandraConfig with default values.
func NewCassandraConfig() CassandraConfig {
	rConf := retries.NewConfig()
	rConf.MaxRetries = 3
	rConf.Backoff.InitialInterval = "1s"
	rConf.Backoff.MaxInterval = "5s"
	rConf.Backoff.MaxElapsedTime = ""

	return CassandraConfig{
		Addresses: []string{},
		TLS:       btls.NewConfig(),
		PasswordAuthenticator: PasswordAuthenticator{
			Enabled:  false,
			Username: "",
			Password: "",
		},
		DisableInitialHostLookup: false,
		Query:                    "",
		ArgsMapping:              "",
		Consistency:              "QUORUM",
		Timeout:                  "600ms",
		ConnectTimeout:           "600ms",
		Config:                   rConf,
		MaxInFlight:              64,
		Batching:                 batchconfig.NewConfig(),
		LoggedBatch:              true,
		UseTokenAwareHostPolicy:  false,
		ShuffleReplicas:          false,
		UseCompressor:            false,
	}
}
