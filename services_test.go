package yadisk

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	client, err := NewClient(
		WithOAuthToken("token"),
		WithBaseURL(ts.URL),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	return client
}

func TestResourcesGetMetaQueryEncoding(t *testing.T) {
	var gotPath string
	var gotQuery string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"path":"disk:/docs"}`)
	})

	limit := 20
	offset := 4
	crop := true
	_, err := client.Resources.GetMeta(context.Background(), ResourceGetRequest{
		Path:        "disk:/docs",
		Fields:      []string{"name", "size"},
		Limit:       &limit,
		Offset:      &offset,
		PreviewCrop: &crop,
		PreviewSize: "M",
		Sort:        "name",
	})
	if err != nil {
		t.Fatalf("GetMeta err: %v", err)
	}

	if gotPath != "/disk/resources" {
		t.Fatalf("path = %s", gotPath)
	}
	want := "fields=name%2Csize&limit=20&offset=4&path=disk%3A%2Fdocs&preview_crop=true&preview_size=M&sort=name"
	if gotQuery != want {
		t.Fatalf("query = %s; want %s", gotQuery, want)
	}
}

func TestEmptyOptionOmitted(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "path=disk%3A%2Fa" {
			t.Fatalf("raw query = %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"path":"disk:/a"}`)
	})

	_, err := client.Resources.GetMeta(context.Background(), ResourceGetRequest{Path: "disk:/a"})
	if err != nil {
		t.Fatalf("GetMeta err: %v", err)
	}
}

func TestServicePathCoverage(t *testing.T) {
	tests := []struct {
		name string
		call func(c *Client) error
		path string
	}{
		{"disk", func(c *Client) error { _, err := c.Disk.Get(context.Background(), DiskGetRequest{}); return err }, "/disk"},
		{"flat", func(c *Client) error {
			_, err := c.Resources.ListAllFiles(context.Background(), FlatFilesRequest{})
			return err
		}, "/disk/resources/files"},
		{"recent", func(c *Client) error {
			_, err := c.Resources.ListRecentUploaded(context.Background(), RecentUploadedRequest{})
			return err
		}, "/disk/resources/last-uploaded"},
		{"create-folder", func(c *Client) error {
			_, err := c.Resources.CreateFolder(context.Background(), CreateFolderRequest{Path: "disk:/a"})
			return err
		}, "/disk/resources"},
		{"upload-url", func(c *Client) error {
			_, err := c.Uploads.GetUploadURL(context.Background(), UploadURLRequest{Path: "disk:/a"})
			return err
		}, "/disk/resources/upload"},
		{"download-url", func(c *Client) error {
			_, err := c.Uploads.GetDownloadURL(context.Background(), DownloadURLRequest{Path: "disk:/a"})
			return err
		}, "/disk/resources/download"},
		{"public", func(c *Client) error {
			_, err := c.Public.GetMeta(context.Background(), PublicResourceRequest{PublicKey: "k"})
			return err
		}, "/disk/public/resources"},
		{"trash", func(c *Client) error { _, err := c.Trash.Empty(context.Background(), TrashDeleteRequest{}); return err }, "/disk/trash/resources"},
		{"ops", func(c *Client) error {
			_, err := c.Operations.GetStatus(context.Background(), OperationStatusRequest{OperationID: "id"})
			return err
		}, "/disk/operations/id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			gotPath := ""
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				gotPath = r.URL.Path
				mu.Unlock()

				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.URL.Path == "/disk":
					fmt.Fprint(w, `{}`)
				case r.URL.Path == "/disk/resources":
					if r.Method == http.MethodPut {
						w.WriteHeader(http.StatusCreated)
					}
					fmt.Fprint(w, `{"href":"x","method":"GET","templated":false}`)
				case r.URL.Path == "/disk/resources/files", r.URL.Path == "/disk/resources/last-uploaded":
					fmt.Fprint(w, `{"items":[]}`)
				case r.URL.Path == "/disk/resources/upload":
					fmt.Fprint(w, `{"href":"u","method":"PUT","templated":false}`)
				case r.URL.Path == "/disk/resources/download":
					fmt.Fprint(w, `{"href":"d","method":"GET","templated":false}`)
				case r.URL.Path == "/disk/public/resources":
					fmt.Fprint(w, `{"public_key":"k"}`)
				case r.URL.Path == "/disk/trash/resources":
					w.WriteHeader(http.StatusNoContent)
				case r.URL.Path == "/disk/operations/id":
					fmt.Fprint(w, `{"status":"success"}`)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			})

			if err := tt.call(client); err != nil {
				t.Fatalf("call err: %v", err)
			}
			if gotPath != tt.path {
				t.Fatalf("path = %s want %s", gotPath, tt.path)
			}
		})
	}
}
