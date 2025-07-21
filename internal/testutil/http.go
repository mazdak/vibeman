package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// NewJSONRequest creates a new HTTP request with JSON body
func NewJSONRequest(method, url string, body interface{}) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// DecodeJSON decodes JSON from a reader
func DecodeJSON(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

// ReadString reads the entire response body as a string
func ReadString(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// ParseErrorResponse parses an error response from the server
func ParseErrorResponse(resp *http.Response) (*ErrorResponse, error) {
	var errResp ErrorResponse
	if err := DecodeJSON(resp.Body, &errResp); err != nil {
		// Try to read as plain text
		body, _ := ReadString(resp.Body)
		errResp.Error = strings.TrimSpace(body)
	}
	return &errResp, nil
}
