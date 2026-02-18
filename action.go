package yadisk

import (
	"net/url"
	"path"
	"strings"
)

func operationRefFromLink(link *Link) *OperationRef {
	if link == nil || link.Href == "" {
		return nil
	}
	u, err := url.Parse(link.Href)
	if err != nil {
		return &OperationRef{Href: link.Href}
	}

	id := u.Query().Get("id")
	if id == "" {
		id = strings.TrimPrefix(path.Base(strings.TrimSuffix(u.Path, "/")), "operations/")
	}
	if id == "" {
		return &OperationRef{Href: link.Href}
	}
	return &OperationRef{ID: id, Href: link.Href}
}

func actionFromStatus(status int, link *Link) ActionResult {
	result := ActionResult{StatusCode: status, Link: link}
	if status == 202 {
		result.Operation = operationRefFromLink(link)
	}
	return result
}
