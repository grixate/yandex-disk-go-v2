package yadisk

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

type PublicService struct {
	client *Client
}

func (s *PublicService) GetMeta(ctx context.Context, req PublicResourceRequest) (*PublicResource, error) {
	if req.PublicKey == "" {
		return nil, errors.New("public_key is required")
	}
	q := url.Values{}
	addString(q, "public_key", req.PublicKey)
	addCSV(q, "fields", req.Fields)
	addInt(q, "limit", req.Limit)
	addInt(q, "offset", req.Offset)
	addString(q, "path", req.Path)
	addBool(q, "preview_crop", req.PreviewCrop)
	addString(q, "preview_size", req.PreviewSize)
	addString(q, "sort", req.Sort)

	out := new(PublicResource)
	_, err := s.client.doJSON(ctx, http.MethodGet, "/disk/public/resources", q, nil, out, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PublicService) GetDownloadURL(ctx context.Context, req PublicDownloadRequest) (*Link, error) {
	if req.PublicKey == "" {
		return nil, errors.New("public_key is required")
	}
	q := url.Values{}
	addString(q, "public_key", req.PublicKey)
	addCSV(q, "fields", req.Fields)
	addString(q, "path", req.Path)

	out := new(Link)
	_, err := s.client.doJSON(ctx, http.MethodGet, "/disk/public/resources/download", q, nil, out, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PublicService) SaveToDisk(ctx context.Context, req PublicSaveRequest) (ActionResult, error) {
	if req.PublicKey == "" {
		return ActionResult{}, errors.New("public_key is required")
	}
	q := url.Values{}
	addString(q, "public_key", req.PublicKey)
	addCSV(q, "fields", req.Fields)
	addBool(q, "force_async", req.ForceAsync)
	addString(q, "name", req.Name)
	addString(q, "path", req.Path)
	addString(q, "save_path", req.SavePath)

	out := new(Link)
	resp, err := s.client.doJSON(ctx, http.MethodPost, "/disk/public/resources/save-to-disk", q, nil, out, http.StatusCreated, http.StatusAccepted)
	if err != nil {
		return ActionResult{}, err
	}
	return actionFromStatus(resp.StatusCode, out), nil
}
