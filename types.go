package yadisk

import (
	"encoding/json"
	"errors"
	"time"
)

type Timestamp struct {
	Time  time.Time
	Raw   string
	Valid bool
}

func (t *Timestamp) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*t = Timestamp{}
		return nil
	}

	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw == "" {
		*t = Timestamp{}
		return nil
	}

	t.Raw = raw
	if ts, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		t.Time = ts
		t.Valid = true
		return nil
	}
	if ts, err := time.Parse(time.RFC3339, raw); err == nil {
		t.Time = ts
		t.Valid = true
		return nil
	}
	return nil
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	if t.Valid {
		return json.Marshal(t.Time.Format(time.RFC3339Nano))
	}
	if t.Raw != "" {
		return json.Marshal(t.Raw)
	}
	return json.Marshal("")
}

type Link struct {
	Href      string `json:"href"`
	Method    string `json:"method"`
	Templated bool   `json:"templated"`
}

type ResourceUploadLink struct {
	Link
	OperationID string `json:"operation_id,omitempty"`
}

type OperationRef struct {
	ID   string
	Href string
}

type ActionResult struct {
	StatusCode int
	Operation  *OperationRef
	Link       *Link
}

type OperationStatus struct {
	Status string `json:"status"`
}

func (s OperationStatus) IsTerminal() bool {
	switch s.Status {
	case "success", "failed", "error", "cancelled":
		return true
	default:
		return false
	}
}

type DiskInfo struct {
	MaxFileSize                int64         `json:"max_file_size"`
	UnlimitedAutouploadEnabled bool          `json:"unlimited_autoupload_enabled"`
	TotalSpace                 int64         `json:"total_space"`
	TrashSize                  int64         `json:"trash_size"`
	IsPaid                     bool          `json:"is_paid"`
	UsedSpace                  int64         `json:"used_space"`
	SystemFolders              SystemFolders `json:"system_folders"`
	User                       User          `json:"user"`
	Revision                   int64         `json:"revision"`
}

type SystemFolders struct {
	Applications  string `json:"applications"`
	Downloads     string `json:"downloads"`
	Google        string `json:"google"`
	Instagram     string `json:"instagram"`
	Mailru        string `json:"mailru"`
	Odnoklassniki string `json:"odnoklassniki"`
	Photostream   string `json:"photostream"`
	Screenshots   string `json:"screenshots"`
	Social        string `json:"social"`
	Vkontakte     string `json:"vkontakte"`
	Facebook      string `json:"facebook"`
}

type User struct {
	Country     string `json:"country"`
	Login       string `json:"login"`
	DisplayName string `json:"display_name"`
	UID         string `json:"uid"`
}

type Owner struct {
	Login       string `json:"login"`
	DisplayName string `json:"display_name"`
	UID         string `json:"uid"`
}

type Share struct {
	IsRoot  bool   `json:"is_root"`
	IsOwned bool   `json:"is_owned"`
	Rights  string `json:"rights"`
}

type Exif struct {
	DateTime string `json:"date_time"`
}

type CommentIDs struct {
	PrivateResource string `json:"private_resource"`
	PublicResource  string `json:"public_resource"`
}

type BaseResource struct {
	ResourceID     string     `json:"resource_id"`
	Share          Share      `json:"share"`
	File           string     `json:"file"`
	Size           int64      `json:"size"`
	PhotosliceTime string     `json:"photoslice_time"`
	Exif           Exif       `json:"exif"`
	MediaType      string     `json:"media_type"`
	SHA256         string     `json:"sha256"`
	Type           string     `json:"type"`
	MimeType       string     `json:"mime_type"`
	Revision       int64      `json:"revision"`
	PublicURL      string     `json:"public_url"`
	Path           string     `json:"path"`
	MD5            string     `json:"md5"`
	PublicKey      string     `json:"public_key"`
	Preview        string     `json:"preview"`
	Name           string     `json:"name"`
	Created        Timestamp  `json:"created"`
	Modified       Timestamp  `json:"modified"`
	CommentIDs     CommentIDs `json:"comment_ids"`
}

type Resource struct {
	BaseResource
	CustomProperties map[string]any `json:"custom_properties"`
	Embedded         Embedded       `json:"_embedded"`
}

type PublicResource struct {
	BaseResource
	ViewsCount int            `json:"views_count"`
	Owner      Owner          `json:"owner"`
	Embedded   PublicEmbedded `json:"_embedded"`
}

type TrashResource struct {
	BaseResource
	Embedded         TrashEmbedded  `json:"_embedded"`
	CustomProperties map[string]any `json:"custom_properties"`
	OriginPath       string         `json:"origin_path"`
	Deleted          Timestamp      `json:"deleted"`
}

type BaseEmbedded struct {
	Sort   string `json:"sort"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Path   string `json:"path"`
	Total  int    `json:"total"`
}

type Embedded struct {
	BaseEmbedded
	Items []Resource `json:"items"`
}

type PublicEmbedded struct {
	BaseEmbedded
	Items []PublicResource `json:"items"`
}

type TrashEmbedded struct {
	BaseEmbedded
	Items []TrashResource `json:"items"`
}

type FilesResourceList struct {
	Items  []Resource `json:"items"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}

type LastUploadedResourceList struct {
	Items []Resource `json:"items"`
	Limit int        `json:"limit"`
}

type PublicResourcesList struct {
	Items  []Resource `json:"items"`
	Type   string     `json:"type"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}

type ResourcePatch struct {
	CustomProperties map[string]any `json:"custom_properties"`
}

// Requests.

type DiskGetRequest struct {
	Fields []string
}

type ResourceGetRequest struct {
	Path        string
	Fields      []string
	Limit       *int
	Offset      *int
	PreviewCrop *bool
	PreviewSize string
	Sort        string
}

func (r ResourceGetRequest) Validate() error {
	if r.Path == "" {
		return errors.New("path is required")
	}
	return nil
}

type FlatFilesRequest struct {
	Fields      []string
	Limit       *int
	MediaType   string
	Offset      *int
	PreviewCrop *bool
	PreviewSize string
	Sort        string
}

type RecentUploadedRequest struct {
	Fields      []string
	Limit       *int
	MediaType   string
	PreviewCrop *bool
	PreviewSize string
}

type RecentPublicRequest struct {
	Fields       []string
	Limit        *int
	Offset       *int
	PreviewCrop  *bool
	PreviewSize  string
	ResourceType string
}

type ResourceUpdateRequest struct {
	Path             string
	Fields           []string
	CustomProperties map[string]any
}

type CreateFolderRequest struct {
	Path   string
	Fields []string
}

type CopyMoveRequest struct {
	From       string
	Path       string
	Fields     []string
	ForceAsync *bool
	Overwrite  *bool
}

type DeleteResourceRequest struct {
	Path        string
	Fields      []string
	ForceAsync  *bool
	MD5         string
	Permanently *bool
}

type PublishRequest struct {
	Path   string
	Fields []string
}

type UploadURLRequest struct {
	Path      string
	Fields    []string
	Overwrite *bool
}

type UploadExternalRequest struct {
	Path             string
	ExternalURL      string
	DisableRedirects *bool
	Fields           []string
}

type DownloadURLRequest struct {
	Path   string
	Fields []string
}

type PublicResourceRequest struct {
	PublicKey   string
	Fields      []string
	Limit       *int
	Offset      *int
	Path        string
	PreviewCrop *bool
	PreviewSize string
	Sort        string
}

type PublicDownloadRequest struct {
	PublicKey string
	Fields    []string
	Path      string
}

type PublicSaveRequest struct {
	PublicKey  string
	Fields     []string
	ForceAsync *bool
	Name       string
	Path       string
	SavePath   string
}

type TrashDeleteRequest struct {
	Fields     []string
	ForceAsync *bool
	Path       string
}

type TrashRestoreRequest struct {
	Path       string
	Fields     []string
	ForceAsync *bool
	Name       string
	Overwrite  *bool
}

type OperationStatusRequest struct {
	OperationID string
	Fields      []string
}

type UploadChunkRequest struct {
	PartSize    int64
	Parallelism int
}
