package input

import (
	"github.com/Jeffail/benthos/v3/internal/docs"
	"github.com/Jeffail/benthos/v3/lib/log"
	"github.com/Jeffail/benthos/v3/lib/metrics"
	"github.com/Jeffail/benthos/v3/lib/types"
)

func init() {
	Constructors[TypeAzureQueueStorage] = TypeSpec{
		constructor: fromSimpleConstructor(func(conf Config, mgr types.Manager, log log.Modular, stats metrics.Type) (Type, error) {
			r, err := newAzureQueueStorage(conf.AzureQueueStorage, log, stats)
			if err != nil {
				return nil, err
			}
			return NewAsyncReader(TypeAzureQueueStorage, false, r, log, stats)
		}),
		Status:  docs.StatusBeta,
		Version: "3.40.0",
		Summary: `
Dequeue objects from an Azure Storage Queue.`,
		Description: `
Dequeue objects from an Azure Storage Queue.

This input adds the following metadata fields to each message:

` + "```" + `
- queue_storage_insertion_time
` + "```" + `

Only one authentication method is required, ` + "`storage_connection_string`" + ` or ` + "`storage_account` and `storage_access_key`" + `. If both are set then the ` + "`storage_connection_string`" + ` is given priority.`,
		FieldSpecs: docs.FieldSpecs{
			docs.FieldCommon(
				"storage_account",
				"The storage account to dequeue messages from. This field is ignored if `storage_connection_string` is set.",
			),
			docs.FieldCommon(
				"storage_access_key",
				"The storage account access key. This field is ignored if `storage_connection_string` is set.",
			),
			docs.FieldCommon(
				"storage_sas_token",
				"The storage account SAS token. This field is ignored if `storage_connection_string` or `storage_access_key` are set.",
			).AtVersion("3.38.0"),
			docs.FieldCommon(
				"storage_connection_string",
				"A storage account connection string. This field is required if `storage_account` and `storage_access_key` / `storage_sas_token` are not set.",
			),
			docs.FieldCommon(
				"queue_name", "The name of the target Storage queue.",
			),
		},
		Categories: []Category{
			CategoryServices,
			CategoryAzure,
		},
	}
}

//------------------------------------------------------------------------------

// AzureQueueStorageConfig contains configuration fields for the AzureQueueStorage
// input type.
type AzureQueueStorageConfig struct {
	StorageAccount          string `json:"storage_account" yaml:"storage_account"`
	StorageAccessKey        string `json:"storage_access_key" yaml:"storage_access_key"`
	StorageSASToken         string `json:"storage_sas_token" yaml:"storage_sas_token"`
	StorageConnectionString string `json:"storage_connection_string" yaml:"storage_connection_string"`
	QueueName               string `json:"queue_name" yaml:"queue_name"`
}

// NewAzureQueueStorageConfig creates a new AzureQueueStorageConfig with default
// values.
func NewAzureQueueStorageConfig() AzureQueueStorageConfig {
	return AzureQueueStorageConfig{}
}
