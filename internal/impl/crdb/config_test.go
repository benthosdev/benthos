package crdb

import (
	"context"
	"testing"

	"github.com/Jeffail/benthos/v3/public/service"
	"github.com/stretchr/testify/require"
)

func TestCRDBConfigParse(t *testing.T) {
	conf := `
crdb_changefeed:
dsn: postgresql://dan:xxxx@free-tier.gcp-us-central1.cockroachlabs.cloud:26257/defaultdb?sslmode=require&options=--cluster%3Dportly-impala-2852
tables:
    - strm_2
options:
    - UPDATED
    - CURSOR='1637953249519902405.0000000000'
`

	spec := crdbChangefeedInputConfig()
	env := service.NewEnvironment()

	selectConfig, err := spec.ParseYAML(conf, env)
	require.NoError(t, err)

	selectInput, err := newCRDBChangefeedInputFromConfig(selectConfig, nil)
	require.NoError(t, err)
	require.NoError(t, selectInput.Close(context.Background()))
}
