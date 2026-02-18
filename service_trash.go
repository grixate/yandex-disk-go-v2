package yadisk

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

type TrashService struct {
	client *Client
}

func (s *TrashService) Empty(ctx context.Context, req TrashDeleteRequest) (ActionResult, error) {
	q := url.Values{}
	addCSV(q, "fields", req.Fields)
	addBool(q, "force_async", req.ForceAsync)
	addString(q, "path", req.Path)

	out := new(Link)
	resp, err := s.client.doJSON(ctx, http.MethodDelete, "/disk/trash/resources", q, nil, out, http.StatusNoContent, http.StatusAccepted)
	if err != nil {
		return ActionResult{}, err
	}
	if resp.StatusCode == http.StatusNoContent {
		return ActionResult{StatusCode: resp.StatusCode}, nil
	}
	return actionFromStatus(resp.StatusCode, out), nil
}

func (s *TrashService) Restore(ctx context.Context, req TrashRestoreRequest) (ActionResult, error) {
	if req.Path == "" {
		return ActionResult{}, errors.New("path is required")
	}
	q := url.Values{}
	addString(q, "path", req.Path)
	addCSV(q, "fields", req.Fields)
	addBool(q, "force_async", req.ForceAsync)
	addString(q, "name", req.Name)
	addBool(q, "overwrite", req.Overwrite)

	out := new(Link)
	resp, err := s.client.doJSON(ctx, http.MethodPut, "/disk/trash/resources/restore", q, nil, out, http.StatusCreated, http.StatusAccepted)
	if err != nil {
		return ActionResult{}, err
	}
	return actionFromStatus(resp.StatusCode, out), nil
}

func (s *TrashService) GetMeta(ctx context.Context, req ResourceGetRequest) (*TrashResource, error) {
	if req.Path == "" {
		return nil, errors.New("path is required")
	}
	q := resourceQuery(req)
	out := new(TrashResource)
	_, err := s.client.doJSON(ctx, http.MethodGet, "/disk/trash/resources", q, nil, out, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return out, nil
}
