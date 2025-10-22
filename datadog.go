package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/docker/docker/daemon/logger"
)

const (
	name = "datadoglogger"
)

type DatadogLogger struct {
	API *datadogV2.LogsApi
	Ctx context.Context
}

func New(info logger.Info) (logger.Logger, error) {
	ctx := context.Background()

	fmt.Printf("ENV: %v\n", info.ContainerEnv)
	fmt.Printf("CONFIG: %v\n", info.Config)

	keys := make(map[string]datadog.APIKey)
	if apiKey, ok := info.Config["DD_API_KEY"]; ok {
		keys["apiKeyAuth"] = datadog.APIKey{Key: apiKey}
	}

	if apiKey, ok := info.Config["DD_APP_KEY"]; ok {
		keys["appKeyAuth"] = datadog.APIKey{Key: apiKey}
	}

	fmt.Printf("AUTH KEYS: %v\n", keys)

	ctx = context.WithValue(
		ctx,
		datadog.ContextAPIKeys,
		keys,
	)

	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)
	api := datadogV2.NewLogsApi(apiClient)

	return &DatadogLogger{
		API: api,
		Ctx: ctx,
	}, nil
}

func (d *DatadogLogger) Name() string { return name }

func (d *DatadogLogger) Log(message *logger.Message) error {
	body := []datadogV2.HTTPLogItem{
		{
			Ddsource: datadog.PtrString("datadog-docker-log-plugin"),
			Ddtags:   datadog.PtrString("env:innovation"),
			Message:  string(message.Line),
		},
	}

	resp, r, err := d.API.SubmitLog(d.Ctx, body, *datadogV2.NewSubmitLogOptionalParameters().WithContentEncoding(datadogV2.CONTENTENCODING_DEFLATE))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `LogsApi.SubmitLog`: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}

	responseContent, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Fprintf(os.Stdout, "Response from `LogsApi.SubmitLog`:\n%s\n", responseContent)

	return nil
}

func (d *DatadogLogger) Close() error { return nil }
