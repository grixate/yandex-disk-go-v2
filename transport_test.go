package yadisk

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDecodeMatrix(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/disk":
			fmt.Fprint(w, `{"total_space":21474836480}`)
		case "/disk/resources":
			if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"href":"https://cloud-api.yandex.net/v1/disk/operations?id=abc","method":"GET","templated":false}`)
		case "/disk/resources/copy":
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"href":"https://cloud-api.yandex.net/v1/disk/operations?id=abc","method":"GET","templated":false}`)
		case "/disk/public/resources/save-to-disk":
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"href":"x","method":"GET","templated":false}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	if _, err := client.Disk.Get(context.Background(), DiskGetRequest{}); err != nil {
		t.Fatalf("disk get err: %v", err)
	}

	res, err := client.Resources.Delete(context.Background(), DeleteResourceRequest{Path: "disk:/a"})
	if err != nil {
		t.Fatalf("delete err: %v", err)
	}
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d", res.StatusCode)
	}

	async, err := client.Resources.Copy(context.Background(), CopyMoveRequest{From: "disk:/a", Path: "disk:/b"})
	if err != nil {
		t.Fatalf("copy err: %v", err)
	}
	if async.Operation == nil || async.Operation.ID != "abc" {
		t.Fatalf("operation = %+v", async.Operation)
	}

	saved, err := client.Public.SaveToDisk(context.Background(), PublicSaveRequest{PublicKey: "k"})
	if err != nil {
		t.Fatalf("save err: %v", err)
	}
	if saved.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d", saved.StatusCode)
	}
}

func TestAPIErrorDecode(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":"UnauthorizedError","message":"token invalid","description":"bad token"}`)
	})

	_, err := client.Disk.Get(context.Background(), DiskGetRequest{})
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("unexpected err type: %T", err)
	}
	if apiErr.Code != "UnauthorizedError" || apiErr.HTTPStatus != http.StatusUnauthorized {
		t.Fatalf("apiErr = %+v", apiErr)
	}
}

func TestContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{}`)
	}))
	defer ts.Close()

	client, err := NewClient(WithOAuthToken("token"), WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = client.Disk.Get(ctx, DiskGetRequest{})
	if err == nil {
		t.Fatal("expected context timeout")
	}
}
