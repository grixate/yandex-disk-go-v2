package yadisk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

const maxUploadPartSize int64 = 10_000_000_000

type UploadsService struct {
	client *Client
}

func (s *UploadsService) GetUploadURL(ctx context.Context, req UploadURLRequest) (*ResourceUploadLink, error) {
	if req.Path == "" {
		return nil, errors.New("path is required")
	}
	q := url.Values{}
	addString(q, "path", req.Path)
	addCSV(q, "fields", req.Fields)
	addBool(q, "overwrite", req.Overwrite)

	out := new(ResourceUploadLink)
	_, err := s.client.doJSON(ctx, http.MethodGet, "/disk/resources/upload", q, nil, out, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *UploadsService) UploadExternal(ctx context.Context, req UploadExternalRequest) (*Link, error) {
	if req.Path == "" || req.ExternalURL == "" {
		return nil, errors.New("path and external_url are required")
	}
	q := url.Values{}
	addString(q, "path", req.Path)
	addString(q, "url", req.ExternalURL)
	addBool(q, "disable_redirects", req.DisableRedirects)
	addCSV(q, "fields", req.Fields)

	out := new(Link)
	_, err := s.client.doJSON(ctx, http.MethodPost, "/disk/resources/upload", q, nil, out, http.StatusAccepted, http.StatusCreated)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *UploadsService) GetDownloadURL(ctx context.Context, req DownloadURLRequest) (*Link, error) {
	if req.Path == "" {
		return nil, errors.New("path is required")
	}
	q := url.Values{}
	addString(q, "path", req.Path)
	addCSV(q, "fields", req.Fields)

	out := new(Link)
	_, err := s.client.doJSON(ctx, http.MethodGet, "/disk/resources/download", q, nil, out, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *UploadsService) OpenDownload(ctx context.Context, req DownloadURLRequest) (io.ReadCloser, error) {
	link, err := s.GetDownloadURL(ctx, req)
	if err != nil {
		return nil, err
	}
	if link.Href == "" {
		return nil, errors.New("empty download href")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, link.Href, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.doRaw(ctx, httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		apiErr := s.client.apiErrorFromResponse(resp, body)
		if err := resp.Body.Close(); err != nil {
			return nil, errors.Join(apiErr, err)
		}
		return nil, apiErr
	}
	return resp.Body, nil
}

func (s *UploadsService) UploadByLink(ctx context.Context, link *ResourceUploadLink, reader io.Reader) (ActionResult, error) {
	if link == nil || link.Href == "" || link.Method == "" {
		return ActionResult{}, errors.New("upload link must have href and method")
	}
	if reader == nil {
		return ActionResult{}, errors.New("reader must not be nil")
	}

	req, err := http.NewRequestWithContext(ctx, link.Method, link.Href, reader)
	if err != nil {
		return ActionResult{}, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := s.client.doRaw(ctx, req)
	if err != nil {
		return ActionResult{}, err
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		apiErr := s.client.apiErrorFromResponse(resp, body)
		if err := resp.Body.Close(); err != nil {
			return ActionResult{}, errors.Join(apiErr, err)
		}
		return ActionResult{}, apiErr
	}
	if err := resp.Body.Close(); err != nil {
		return ActionResult{}, err
	}

	result := ActionResult{StatusCode: resp.StatusCode}
	if resp.StatusCode == http.StatusAccepted {
		result.Operation = &OperationRef{ID: link.OperationID, Href: link.Href}
	}
	return result, nil
}

func (s *UploadsService) UploadInChunks(ctx context.Context, link *ResourceUploadLink, reader io.ReadSeeker, cfg UploadChunkRequest) (ActionResult, error) {
	if link == nil || link.Href == "" || link.Method == "" {
		return ActionResult{}, errors.New("upload link must have href and method")
	}
	if reader == nil {
		return ActionResult{}, errors.New("reader must not be nil")
	}
	partSize := cfg.PartSize
	if partSize <= 0 {
		partSize = 10 * 1024 * 1024
	}
	if partSize > maxUploadPartSize {
		partSize = maxUploadPartSize
	}

	total, err := reader.Seek(0, io.SeekEnd)
	if err != nil {
		return ActionResult{}, err
	}
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return ActionResult{}, err
	}

	buf := make([]byte, partSize)
	var start int64
	for start < total {
		select {
		case <-ctx.Done():
			return ActionResult{}, ctx.Err()
		default:
		}

		remaining := total - start
		chunkSize := int64(len(buf))
		if remaining < chunkSize {
			chunkSize = remaining
		}

		n, err := io.ReadFull(reader, buf[:chunkSize])
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			return ActionResult{}, err
		}
		if n == 0 {
			break
		}

		end := start + int64(n) - 1
		req, err := http.NewRequestWithContext(ctx, link.Method, link.Href, io.NopCloser(bytesReader(buf[:n])))
		if err != nil {
			return ActionResult{}, err
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Content-Range", "bytes "+strconv.FormatInt(start, 10)+"-"+strconv.FormatInt(end, 10)+"/"+strconv.FormatInt(total, 10))

		resp, err := s.client.doRaw(ctx, req)
		if err != nil {
			return ActionResult{}, err
		}
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
			body, _ := io.ReadAll(resp.Body)
			apiErr := s.client.apiErrorFromResponse(resp, body)
			if err := resp.Body.Close(); err != nil {
				return ActionResult{}, errors.Join(apiErr, err)
			}
			return ActionResult{}, apiErr
		}
		if err := resp.Body.Close(); err != nil {
			return ActionResult{}, err
		}
		start += int64(n)
	}

	return ActionResult{StatusCode: http.StatusAccepted, Operation: &OperationRef{ID: link.OperationID, Href: link.Href}}, nil
}

func bytesReader(p []byte) io.Reader {
	return &sliceReader{buf: p}
}

type sliceReader struct {
	buf []byte
	off int
}

func (r *sliceReader) Read(p []byte) (int, error) {
	if r.off >= len(r.buf) {
		return 0, io.EOF
	}
	n := copy(p, r.buf[r.off:])
	r.off += n
	return n, nil
}

func (s *UploadsService) ValidatePartSize(partSize int64) error {
	if partSize <= 0 {
		return errors.New("part size must be > 0")
	}
	if partSize > maxUploadPartSize {
		return fmt.Errorf("part size must be <= %d", maxUploadPartSize)
	}
	return nil
}
