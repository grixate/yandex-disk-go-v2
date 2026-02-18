# yandex-disk-go v2

Typed Go SDK for Yandex Disk REST API.

## Install

```bash
go get github.com/grixate/yandex-disk-go-v2
```

## Quick Start

```go
package main

import (
	"context"
	"log"

	"github.com/grixate/yandex-disk-go-v2"
)

func main() {
	client, err := yadisk.NewClient(
		yadisk.WithOAuthToken("YANDEX_OAUTH_TOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close(context.Background())

	disk, err := client.Disk.Get(context.Background(), yadisk.DiskGetRequest{Fields: []string{"total_space", "used_space"}})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("used=%d total=%d", disk.UsedSpace, disk.TotalSpace)
}
```

## Services

- `Client.Disk`
- `Client.Resources`
- `Client.Uploads`
- `Client.Public`
- `Client.Trash`
- `Client.Operations`
- `Client.Worker`

## Integration tests

Integration tests are opt-in:

```bash
YANDEX_TOKEN=... go test -run Integration -v ./...
```
