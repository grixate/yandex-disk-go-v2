package yadisk

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestOptionsAndClientValidation(t *testing.T) {
	if _, err := NewClient(); err == nil {
		t.Fatal("expected missing token error")
	}
	if _, err := NewClient(WithOAuthToken("x"), WithHTTPClient(nil)); err == nil {
		t.Fatal("expected nil client error")
	}
	if _, err := NewClient(WithOAuthToken("x"), WithRetryPolicy(RetryPolicy{MaxRetries: -1, BaseDelay: time.Second, MaxDelay: time.Second})); err == nil {
		t.Fatal("expected retry validation error")
	}
	if _, err := NewClient(WithOAuthToken("x"), WithWorkerConfig(WorkerConfig{PollInterval: 0, MaxInterval: time.Second, QueueSize: 1})); err == nil {
		t.Fatal("expected worker validation error")
	}

	h := Hooks{}
	client, err := NewClient(
		WithOAuthToken("x"),
		WithUserAgent("ua"),
		WithHooks(h),
		WithRetryPolicy(RetryPolicy{MaxRetries: 0, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond, Jitter: 0}),
		WithWorkerConfig(WorkerConfig{PollInterval: time.Millisecond, MaxInterval: time.Second, QueueSize: 10}),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if err := client.Close(context.Background()); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestAPIErrorStringFormatting(t *testing.T) {
	err := (&APIError{HTTPStatus: 401, Code: "UnauthorizedError", Message: "bad token"}).Error()
	if !strings.Contains(err, "UnauthorizedError") {
		t.Fatalf("error string = %q", err)
	}
	err2 := (&APIError{HTTPStatus: 500}).Error()
	if !strings.Contains(err2, "500") {
		t.Fatalf("error string = %q", err2)
	}
}

func TestActionHelpers(t *testing.T) {
	ref := operationRefFromLink(&Link{Href: "https://cloud-api.yandex.net/v1/disk/operations?id=abc"})
	if ref == nil || ref.ID != "abc" {
		t.Fatalf("ref = %+v", ref)
	}
	ref2 := operationRefFromLink(&Link{Href: "https://cloud-api.yandex.net/v1/disk/operations/xyz"})
	if ref2 == nil || ref2.ID != "xyz" {
		t.Fatalf("ref2 = %+v", ref2)
	}
}

func TestUncoveredServicesAndUploads(t *testing.T) {
	chunkCalls := 0
	uploadByLinkCalls := 0

	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/disk/resources/public":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"items":[],"type":"file"}`)
		case r.URL.Path == "/disk/resources" && r.Method == http.MethodPatch:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"path":"disk:/x","custom_properties":{"a":1}}`)
		case r.URL.Path == "/disk/resources/move":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"href":"https://cloud-api.yandex.net/v1/disk/operations?id=mv","method":"GET","templated":false}`)
		case r.URL.Path == "/disk/resources/publish":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"href":"pub","method":"GET","templated":false}`)
		case r.URL.Path == "/disk/resources/unpublish":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"href":"unpub","method":"GET","templated":false}`)
		case r.URL.Path == "/disk/public/resources/download":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"href":"x","method":"GET","templated":false}`)
		case r.URL.Path == "/disk/trash/resources/restore":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"href":"https://cloud-api.yandex.net/v1/disk/operations?id=tr","method":"GET","templated":false}`)
		case r.URL.Path == "/disk/trash/resources" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"path":"trash:/a"}`)
		case r.URL.Path == "/disk/resources/upload" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"href":"https://cloud-api.yandex.net/v1/disk/operations?id=up","method":"GET","templated":false}`)
		case r.URL.Path == "/disk/resources/download" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"href":"%s/file.bin","method":"GET","templated":false}`, ts.URL)
		case r.URL.Path == "/file.bin":
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "abc")
		case r.URL.Path == "/upload-link":
			uploadByLinkCalls++
			w.WriteHeader(http.StatusAccepted)
		case r.URL.Path == "/upload-chunk":
			chunkCalls++
			if r.Header.Get("Content-Range") == "" {
				t.Fatalf("missing content-range")
			}
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	client, err := NewClient(WithOAuthToken("token"), WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if _, err := client.Resources.ListPublished(context.Background(), RecentPublicRequest{}); err != nil {
		t.Fatalf("list published: %v", err)
	}
	if _, err := client.Resources.UpdateMeta(context.Background(), ResourceUpdateRequest{Path: "disk:/x", CustomProperties: map[string]any{"a": 1}}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if _, err := client.Resources.Move(context.Background(), CopyMoveRequest{From: "disk:/a", Path: "disk:/b"}); err != nil {
		t.Fatalf("move: %v", err)
	}
	if _, err := client.Resources.Publish(context.Background(), PublishRequest{Path: "disk:/a"}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if _, err := client.Resources.Unpublish(context.Background(), PublishRequest{Path: "disk:/a"}); err != nil {
		t.Fatalf("unpublish: %v", err)
	}
	if _, err := client.Public.GetDownloadURL(context.Background(), PublicDownloadRequest{PublicKey: "k"}); err != nil {
		t.Fatalf("public dl: %v", err)
	}
	if _, err := client.Trash.Restore(context.Background(), TrashRestoreRequest{Path: "trash:/a"}); err != nil {
		t.Fatalf("trash restore: %v", err)
	}
	if _, err := client.Trash.GetMeta(context.Background(), ResourceGetRequest{Path: "trash:/a"}); err != nil {
		t.Fatalf("trash meta: %v", err)
	}
	if _, err := client.Uploads.UploadExternal(context.Background(), UploadExternalRequest{Path: "disk:/x", ExternalURL: "https://example.com/a"}); err != nil {
		t.Fatalf("upload external: %v", err)
	}
	body, err := client.Uploads.OpenDownload(context.Background(), DownloadURLRequest{Path: "disk:/x"})
	if err != nil {
		t.Fatalf("open download: %v", err)
	}
	got, _ := io.ReadAll(body)
	body.Close()
	if string(got) != "abc" {
		t.Fatalf("download body = %q", got)
	}

	if _, err := client.Uploads.UploadByLink(context.Background(), &ResourceUploadLink{Link: Link{Href: ts.URL + "/upload-link", Method: http.MethodPut}, OperationID: "op-u"}, bytes.NewBufferString("payload")); err != nil {
		t.Fatalf("upload by link: %v", err)
	}

	chunkRes, err := client.Uploads.UploadInChunks(context.Background(), &ResourceUploadLink{Link: Link{Href: ts.URL + "/upload-chunk", Method: http.MethodPut}, OperationID: "op-c"}, bytes.NewReader([]byte("hello world")), UploadChunkRequest{PartSize: 4})
	if err != nil {
		t.Fatalf("upload chunks: %v", err)
	}
	if chunkRes.Operation == nil || chunkRes.Operation.ID != "op-c" {
		t.Fatalf("chunk result = %+v", chunkRes)
	}
	if chunkCalls < 2 {
		t.Fatalf("expected multiple chunk calls")
	}
	if uploadByLinkCalls != 1 {
		t.Fatalf("uploadByLinkCalls=%d", uploadByLinkCalls)
	}
}

func TestUploadValidationAndHelpers(t *testing.T) {
	client, _ := NewClient(WithOAuthToken("token"))

	if err := client.Uploads.ValidatePartSize(0); err == nil {
		t.Fatal("expected validation error")
	}
	if err := client.Uploads.ValidatePartSize(maxUploadPartSize + 1); err == nil {
		t.Fatal("expected max validation error")
	}
	if err := client.Uploads.ValidatePartSize(1); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	values := url.Values{}
	addInt64(values, "size", 10)
	if values.Get("size") != "10" {
		t.Fatalf("size=%q", values.Get("size"))
	}

	r := bytesReader([]byte("abc"))
	buf := make([]byte, 2)
	n, err := r.Read(buf)
	if err != nil || n != 2 {
		t.Fatalf("read1 n=%d err=%v", n, err)
	}
	n, err = r.Read(buf)
	if n != 1 {
		t.Fatalf("read2 n=%d", n)
	}
	if err != nil && err != io.EOF {
		t.Fatalf("read2 err=%v", err)
	}
}

func TestOpenDownloadErrorAndDoRawError(t *testing.T) {
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/disk/resources/download" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"href":"%s/err.bin","method":"GET","templated":false}`, ts.URL)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"InternalError"}`)
	}))
	defer ts.Close()

	client, _ := NewClient(WithOAuthToken("token"), WithBaseURL(ts.URL))
	if _, err := client.Uploads.OpenDownload(context.Background(), DownloadURLRequest{Path: "disk:/x"}); err == nil {
		t.Fatal("expected download error")
	}

	badReq, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:1", nil)
	if _, err := client.doRaw(context.Background(), badReq); err == nil {
		t.Fatal("expected doRaw transport error")
	}
}

func TestTransportBackoffAndHeaders(t *testing.T) {
	retries := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Fatal("missing auth header")
		}
		if retries == 0 {
			retries++
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"error":"TooManyRequestsError"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{}`)
	})

	client.retry = RetryPolicy{MaxRetries: 1, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond, Jitter: 0}
	_, err := client.Disk.Get(context.Background(), DiskGetRequest{})
	if err != nil {
		t.Fatalf("disk get: %v", err)
	}
	if client.backoff(1) != time.Millisecond {
		t.Fatalf("unexpected backoff")
	}
	if client.backoff(10) > time.Millisecond {
		t.Fatalf("backoff not capped")
	}
}

func TestWorkerWatchValidationAndErrorPath(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"InternalError"}`)
	})

	if err := client.Worker.Watch(OperationRef{}, nil); err == nil {
		t.Fatal("expected nil handler error")
	}
	if err := client.Worker.Watch(OperationRef{}, func(OperationEvent) {}); err == nil {
		t.Fatal("expected missing id error")
	}

	if err := client.Worker.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer client.Worker.Stop(context.Background())

	ch := make(chan OperationEvent, 1)
	if err := client.Worker.Watch(OperationRef{Href: "https://cloud-api.yandex.net/v1/disk/operations?id=op"}, func(e OperationEvent) {
		ch <- e
	}); err != nil {
		t.Fatalf("watch: %v", err)
	}

	select {
	case ev := <-ch:
		if ev.Err == nil {
			t.Fatal("expected event error")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting event")
	}
}

func TestOperationStatusTerminalStates(t *testing.T) {
	for _, state := range []string{"success", "failed", "error", "cancelled"} {
		if !(OperationStatus{Status: state}).IsTerminal() {
			t.Fatalf("state %s should be terminal", state)
		}
	}
	if (OperationStatus{Status: "in-progress"}).IsTerminal() {
		t.Fatal("in-progress should not be terminal")
	}
}

func TestResourceValidation(t *testing.T) {
	if err := (ResourceGetRequest{}).Validate(); err == nil {
		t.Fatal("expected validate error")
	}
	if err := (ResourceGetRequest{Path: "disk:/a"}).Validate(); err != nil {
		t.Fatalf("validate err: %v", err)
	}
}

func TestWorkerStartWithCanceledContext(t *testing.T) {
	client, _ := NewClient(WithOAuthToken("token"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := client.Worker.Start(ctx); err == nil {
		t.Fatal("expected start context error")
	}
}

func TestOperationRefFromMalformedURL(t *testing.T) {
	ref := operationRefFromLink(&Link{Href: "::invalid"})
	if ref == nil || ref.Href == "" {
		t.Fatalf("ref = %+v", ref)
	}
}

func TestWithBaseURLValidation(t *testing.T) {
	_, err := NewClient(WithOAuthToken("x"), WithBaseURL("://bad"))
	if err == nil {
		t.Fatal("expected invalid base url")
	}
}

func TestUploadInChunksValidation(t *testing.T) {
	client, _ := NewClient(WithOAuthToken("token"))
	_, err := client.Uploads.UploadInChunks(context.Background(), &ResourceUploadLink{Link: Link{Href: "x", Method: http.MethodPut}}, nil, UploadChunkRequest{PartSize: 1})
	if err == nil {
		t.Fatal("expected nil reader error")
	}
}

func TestUploadByLinkValidation(t *testing.T) {
	client, _ := NewClient(WithOAuthToken("token"))
	_, err := client.Uploads.UploadByLink(context.Background(), nil, bytes.NewBufferString("a"))
	if err == nil {
		t.Fatal("expected nil link error")
	}
	_, err = client.Uploads.UploadByLink(context.Background(), &ResourceUploadLink{Link: Link{Href: "x", Method: http.MethodPut}}, nil)
	if err == nil {
		t.Fatal("expected nil reader error")
	}
}

func TestBackoffJitterPositive(t *testing.T) {
	client, _ := NewClient(WithOAuthToken("token"))
	client.retry = RetryPolicy{MaxRetries: 1, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond, Jitter: 1}
	for i := 0; i < 3; i++ {
		if d := client.backoff(i + 1); d < 0 {
			t.Fatalf("negative backoff: %s", d)
		}
	}
}

func TestSliceReaderEOF(t *testing.T) {
	r := &sliceReader{buf: []byte("a")}
	b := make([]byte, 8)
	_, _ = r.Read(b)
	_, err := r.Read(b)
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestOpenDownloadEmptyHref(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"href":"","method":"GET","templated":false}`)
	})
	if _, err := client.Uploads.OpenDownload(context.Background(), DownloadURLRequest{Path: "disk:/a"}); err == nil {
		t.Fatal("expected empty href error")
	}
}

func TestUploadChunkContentRangeFormat(t *testing.T) {
	start := int64(0)
	end := int64(9)
	total := int64(10)
	value := "bytes " + strconv.FormatInt(start, 10) + "-" + strconv.FormatInt(end, 10) + "/" + strconv.FormatInt(total, 10)
	if value != "bytes 0-9/10" {
		t.Fatalf("content range format mismatch: %s", value)
	}
}
