package yadisk

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

type ResourcesService struct {
	client *Client
}

func (s *ResourcesService) GetMeta(ctx context.Context, req ResourceGetRequest) (*Resource, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	q := resourceQuery(req)

	out := new(Resource)
	_, err := s.client.doJSON(ctx, http.MethodGet, "/disk/resources", q, nil, out, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ResourcesService) ListAllFiles(ctx context.Context, req FlatFilesRequest) (*FilesResourceList, error) {
	q := url.Values{}
	addCSV(q, "fields", req.Fields)
	addInt(q, "limit", req.Limit)
	addString(q, "media_type", req.MediaType)
	addInt(q, "offset", req.Offset)
	addBool(q, "preview_crop", req.PreviewCrop)
	addString(q, "preview_size", req.PreviewSize)
	addString(q, "sort", req.Sort)

	out := new(FilesResourceList)
	_, err := s.client.doJSON(ctx, http.MethodGet, "/disk/resources/files", q, nil, out, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ResourcesService) ListRecentUploaded(ctx context.Context, req RecentUploadedRequest) (*LastUploadedResourceList, error) {
	q := url.Values{}
	addCSV(q, "fields", req.Fields)
	addInt(q, "limit", req.Limit)
	addString(q, "media_type", req.MediaType)
	addBool(q, "preview_crop", req.PreviewCrop)
	addString(q, "preview_size", req.PreviewSize)

	out := new(LastUploadedResourceList)
	_, err := s.client.doJSON(ctx, http.MethodGet, "/disk/resources/last-uploaded", q, nil, out, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ResourcesService) ListPublished(ctx context.Context, req RecentPublicRequest) (*PublicResourcesList, error) {
	q := url.Values{}
	addCSV(q, "fields", req.Fields)
	addInt(q, "limit", req.Limit)
	addInt(q, "offset", req.Offset)
	addBool(q, "preview_crop", req.PreviewCrop)
	addString(q, "preview_size", req.PreviewSize)
	addString(q, "type", req.ResourceType)

	out := new(PublicResourcesList)
	_, err := s.client.doJSON(ctx, http.MethodGet, "/disk/resources/public", q, nil, out, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ResourcesService) UpdateMeta(ctx context.Context, req ResourceUpdateRequest) (*Resource, error) {
	if req.Path == "" {
		return nil, errors.New("path is required")
	}
	q := url.Values{}
	addString(q, "path", req.Path)
	addCSV(q, "fields", req.Fields)

	out := new(Resource)
	payload := ResourcePatch{CustomProperties: req.CustomProperties}
	_, err := s.client.doJSON(ctx, http.MethodPatch, "/disk/resources", q, payload, out, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ResourcesService) CreateFolder(ctx context.Context, req CreateFolderRequest) (*Link, error) {
	if req.Path == "" {
		return nil, errors.New("path is required")
	}
	q := url.Values{}
	addString(q, "path", req.Path)
	addCSV(q, "fields", req.Fields)

	out := new(Link)
	_, err := s.client.doJSON(ctx, http.MethodPut, "/disk/resources", q, nil, out, http.StatusCreated, http.StatusConflict)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ResourcesService) Copy(ctx context.Context, req CopyMoveRequest) (ActionResult, error) {
	return s.copyOrMove(ctx, "/disk/resources/copy", req)
}

func (s *ResourcesService) Move(ctx context.Context, req CopyMoveRequest) (ActionResult, error) {
	return s.copyOrMove(ctx, "/disk/resources/move", req)
}

func (s *ResourcesService) Delete(ctx context.Context, req DeleteResourceRequest) (ActionResult, error) {
	if req.Path == "" {
		return ActionResult{}, errors.New("path is required")
	}
	q := url.Values{}
	addString(q, "path", req.Path)
	addCSV(q, "fields", req.Fields)
	addBool(q, "force_async", req.ForceAsync)
	addString(q, "md5", req.MD5)
	addBool(q, "permanently", req.Permanently)

	out := new(Link)
	resp, err := s.client.doJSON(ctx, http.MethodDelete, "/disk/resources", q, nil, out, http.StatusNoContent, http.StatusAccepted)
	if err != nil {
		return ActionResult{}, err
	}
	if resp.StatusCode == http.StatusNoContent {
		return ActionResult{StatusCode: resp.StatusCode}, nil
	}
	return actionFromStatus(resp.StatusCode, out), nil
}

func (s *ResourcesService) Publish(ctx context.Context, req PublishRequest) (*Link, error) {
	return s.publishAction(ctx, "/disk/resources/publish", req)
}

func (s *ResourcesService) Unpublish(ctx context.Context, req PublishRequest) (*Link, error) {
	return s.publishAction(ctx, "/disk/resources/unpublish", req)
}

func (s *ResourcesService) copyOrMove(ctx context.Context, endpoint string, req CopyMoveRequest) (ActionResult, error) {
	if req.From == "" || req.Path == "" {
		return ActionResult{}, errors.New("from and path are required")
	}
	q := url.Values{}
	addString(q, "from", req.From)
	addString(q, "path", req.Path)
	addCSV(q, "fields", req.Fields)
	addBool(q, "force_async", req.ForceAsync)
	addBool(q, "overwrite", req.Overwrite)

	out := new(Link)
	resp, err := s.client.doJSON(ctx, http.MethodPost, endpoint, q, nil, out, http.StatusCreated, http.StatusAccepted)
	if err != nil {
		return ActionResult{}, err
	}
	return actionFromStatus(resp.StatusCode, out), nil
}

func (s *ResourcesService) publishAction(ctx context.Context, endpoint string, req PublishRequest) (*Link, error) {
	if req.Path == "" {
		return nil, errors.New("path is required")
	}
	q := url.Values{}
	addString(q, "path", req.Path)
	addCSV(q, "fields", req.Fields)

	out := new(Link)
	_, err := s.client.doJSON(ctx, http.MethodPut, endpoint, q, nil, out, http.StatusOK, http.StatusCreated)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func resourceQuery(req ResourceGetRequest) url.Values {
	q := url.Values{}
	addString(q, "path", req.Path)
	addCSV(q, "fields", req.Fields)
	addInt(q, "limit", req.Limit)
	addInt(q, "offset", req.Offset)
	addBool(q, "preview_crop", req.PreviewCrop)
	addString(q, "preview_size", req.PreviewSize)
	addString(q, "sort", req.Sort)
	return q
}
