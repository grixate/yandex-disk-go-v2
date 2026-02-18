# Migration: v1 -> v2

## Major breaking changes

1. Constructor changed from `NewYaDisk(ctx, httpClient, token)` to `NewClient(opts...)`.
2. Methods now require `context.Context` explicitly.
3. Parameter lists moved to typed request structs.
4. Async operations return `ActionResult`/`OperationRef` and can be observed by `Client.Worker`.
5. Numeric fields use `int64` in response models.
6. API errors are returned as `*APIError`.

## Common mapping

| v1 | v2 |
|---|---|
| `NewYaDisk(ctx, client, token)` | `NewClient(WithOAuthToken(...), WithHTTPClient(...))` |
| `GetDisk(fields)` | `client.Disk.Get(ctx, DiskGetRequest{Fields: fields})` |
| `GetResource(path, fields, ...)` | `client.Resources.GetMeta(ctx, ResourceGetRequest{...})` |
| `UpdateResource(path, fields, body)` | `client.Resources.UpdateMeta(ctx, ResourceUpdateRequest{...})` |
| `GetResourceUploadLink(...)` | `client.Uploads.GetUploadURL(ctx, UploadURLRequest{...})` |
| `PerformUpload(link, buf)` | `client.Uploads.UploadByLink(ctx, link, reader)` |
| `PerformPartialUpload(...)` | `client.Uploads.UploadInChunks(ctx, link, seeker, UploadChunkRequest{...})` |
| `GetOperationStatus(id, fields)` | `client.Operations.GetStatus(ctx, OperationStatusRequest{...})` |

## Async worker example

```go
worker := client.Worker
_ = worker.Start(ctx)
_ = worker.Watch(yadisk.OperationRef{ID: opID}, func(e yadisk.OperationEvent) {
    if e.Done {
        // terminal state
    }
})
```
