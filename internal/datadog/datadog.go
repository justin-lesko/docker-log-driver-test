package datadog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/docker/docker/daemon/logger"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

const (
	name = "datadoglogger"
)

type DatadogLogger struct {
	API  *datadogV2.LogsApi
	Ctx  context.Context
	Info logger.Info
}

func New(info logger.Info) (logger.Logger, error) {
	ctx := context.Background()

	keys := make(map[string]datadog.APIKey)
	if apiKey, ok := info.Config["DD_API_KEY"]; ok {
		keys["apiKeyAuth"] = datadog.APIKey{Key: apiKey}
	}

	if apiKey, ok := info.Config["DD_APP_KEY"]; ok {
		keys["appKeyAuth"] = datadog.APIKey{Key: apiKey}
	}

	ctx = context.WithValue(
		ctx,
		datadog.ContextAPIKeys,
		keys,
	)

	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)
	api := datadogV2.NewLogsApi(apiClient)

	return &DatadogLogger{
		API:  api,
		Ctx:  ctx,
		Info: info,
	}, nil
}

func (d *DatadogLogger) Name() string { return name }

func (d *DatadogLogger) Log(message *logger.Message) error {
	tags := d.GetContainerTags(d.Info)
	fmt.Printf("EXTRACED CONTAINER TAGS: %s\n", tags)

	body := []datadogV2.HTTPLogItem{
		{
			Ddsource: datadog.PtrString("ecs"),
			Ddtags:   datadog.PtrString(tags),
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

func (d *DatadogLogger) GetContainerTags(info logger.Info) string {
	tags := []string{}

	if info.ContainerID != "" {
		tags = append(tags, fmt.Sprintf("container_id:%s", info.ContainerID))
	}
	if info.ContainerName != "" {
		tags = append(tags, fmt.Sprintf("container_name:%s", strings.TrimPrefix(info.ContainerName, "/")))
	}
	if info.ContainerImageName != "" {
		tags = append(tags, fmt.Sprintf("image_name:%s", info.ContainerImageName))
	}
	if info.ContainerImageID != "" {
		tags = append(tags, fmt.Sprintf("image_id:%s", info.ContainerImageID))
	}

	for key, value := range info.ContainerLabels {
		if value != "" {
			sanitizedKey := strings.TrimPrefix(key, "com.amazonaws.ecs.")
			tags = append(tags, fmt.Sprintf("%s:%s", sanitizedKey, value))
		}
	}

	// TODO: remove
	tags = append(tags, "env:innovation")

	return strings.Join(tags, ",")
}
