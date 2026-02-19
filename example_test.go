package yadisk_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/grixate/yandex-disk-go-v2"
)

func ExampleClient_Disk() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/disk" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"total_space":10,"used_space":3}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	client, _ := yadisk.NewClient(
		yadisk.WithOAuthToken("token"),
		yadisk.WithBaseURL(ts.URL),
	)
	disk, _ := client.Disk.Get(context.Background(), yadisk.DiskGetRequest{})
	fmt.Println(disk.TotalSpace, disk.UsedSpace)
	// Output: 10 3
}
