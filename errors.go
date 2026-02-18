package yadisk

import "fmt"

type APIError struct {
	HTTPStatus  int
	Code        string `json:"error"`
	Message     string `json:"message"`
	Description string `json:"description"`
	RequestID   string
	RawBody     []byte
}

func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Code != "" && e.Message != "" {
		return fmt.Sprintf("yadisk api error %d %s: %s", e.HTTPStatus, e.Code, e.Message)
	}
	if e.Code != "" {
		return fmt.Sprintf("yadisk api error %d %s", e.HTTPStatus, e.Code)
	}
	if e.Message != "" {
		return fmt.Sprintf("yadisk api error %d: %s", e.HTTPStatus, e.Message)
	}
	return fmt.Sprintf("yadisk api error %d", e.HTTPStatus)
}
