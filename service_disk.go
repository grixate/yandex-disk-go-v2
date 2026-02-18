package yadisk

import (
	"context"
	"net/http"
	"net/url"
)

type DiskService struct {
	client *Client
}

func (s *DiskService) Get(ctx context.Context, req DiskGetRequest) (*DiskInfo, error) {
	q := url.Values{}
	addCSV(q, "fields", req.Fields)

	out := new(DiskInfo)
	_, err := s.client.doJSON(ctx, http.MethodGet, "/disk", q, nil, out, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return out, nil
}
