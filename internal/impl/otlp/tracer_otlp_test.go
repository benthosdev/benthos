package otlp

import (
	"testing"

	"github.com/benthosdev/benthos/v4/internal/cli"
	"github.com/benthosdev/benthos/v4/public/service"
	"github.com/stretchr/testify/assert"
)

func TestSuccessfulParsingOfNonDefaultValues(t *testing.T) {
	config := `
http:
  - url: otlp.http
grpc:
  - url: otlp.grpc
tags: {
  "1": "2",
	"a":"b"
}
`
	parsed := parseValidYAML(config, t)

	httpCollectors, err := collectors(parsed, "http")
	assert.Nil(t, err, "should be able to retrieve the value")
	grpcCollectors, err := collectors(parsed, "grpc")
	assert.Nil(t, err, "should be able to retrieve the value")
	tags, err := parsed.FieldStringMap("tags")
	assert.Nil(t, err, "should be able to retrieve the value")

	assert.Equal(t, []collector{{url: "otlp.http"}}, httpCollectors)
	assert.Equal(t, []collector{{url: "otlp.grpc"}}, grpcCollectors)
	assert.Equal(t, map[string]string{
		"1": "2",
		"a": "b",
	}, tags)
}

func TestSuccessfulParsingOfEmptyYAML(t *testing.T) {
	config := ``

	parsed := parseValidYAML(config, t)

	httpUrls, err := parsed.FieldObjectList("http")
	assert.Nil(t, err, "should be able to retrieve the value")
	assert.Len(t, httpUrls, 0, "currently the default values for this field don't work")
	grpcUrls, err := parsed.FieldObjectList("grpc")
	assert.Nil(t, err, "should be able to retrieve the value")
	assert.Len(t, grpcUrls, 0, "currently the default values for this field don't work")

	tags, err := parsed.FieldStringMap("tags")
	assert.Nil(t, err, "should be able to retrieve the value")
	assert.Equal(t, map[string]string{
		"service.name":    "benthos",
		"service.version": cli.Version,
	}, tags)
}

// Somehow this way the url's default values get filled, probably a bug
func TestParsingOfEmptyUrlValues(t *testing.T) {
	config := `
http:
 -
grpc:
 -
`
	parsed := parseValidYAML(config, t)

	httpUrls, err := parsed.FieldObjectList("http")
	assert.Nil(t, err, "should be able to retrieve the value")
	assert.Len(t, httpUrls, 1, "only 1 default value is expected")
	def, err := httpUrls[0].FieldString("url")
	assert.Nil(t, err, "should be able to retrieve the value")
	assert.Equal(t, "localhost:4318", def)
	grpcUrls, err := parsed.FieldObjectList("grpc")
	assert.Nil(t, err, "should be able to retrieve the value")
	assert.Len(t, grpcUrls, 1, "only 1 default value is expected")
	def, err = grpcUrls[0].FieldString("url")
	assert.Nil(t, err, "should be able to retrieve the value")
	assert.Equal(t, "localhost:4317", def)

	tags, err := parsed.FieldStringMap("tags")
	assert.Nil(t, err, "should be able to retrieve the value")
	assert.Equal(t, map[string]string{
		"service.name":    "benthos",
		"service.version": cli.Version,
	}, tags)
}

func parseValidYAML(config string, t *testing.T) *service.ParsedConfig {
	env := service.NewEnvironment()
	parsed, err := configSpec.ParseYAML(config, env)
	assert.Nil(t, err, "the yaml provided should be valid")
	return parsed
}
