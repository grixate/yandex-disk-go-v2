package yadisk

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

type OperationsService struct {
	client *Client
}

func (s *OperationsService) GetStatus(ctx context.Context, req OperationStatusRequest) (*OperationStatus, error) {
	if req.OperationID == "" {
		return nil, errors.New("operation_id is required")
	}
	q := url.Values{}
	addCSV(q, "fields", req.Fields)

	out := new(OperationStatus)
	_, err := s.client.doJSON(ctx, http.MethodGet, "/disk/operations/"+req.OperationID, q, nil, out, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return out, nil
}
