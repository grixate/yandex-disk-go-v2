package yadisk

import (
	"context"
	"os"
	"testing"
)

func TestIntegrationDiskSmoke(t *testing.T) {
	token := os.Getenv("YANDEX_TOKEN")
	if token == "" {
		t.Skip("YANDEX_TOKEN is not set")
	}
	if os.Getenv("RUN_INTEGRATION") == "" {
		t.Skip("RUN_INTEGRATION is not set")
	}

	client, err := NewClient(WithOAuthToken(token))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Disk.Get(context.Background(), DiskGetRequest{Fields: []string{"total_space", "used_space"}})
	if err != nil {
		t.Fatalf("disk get: %v", err)
	}
}
