package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVersion(t *testing.T) {
	writer := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/version", nil)
	if err != nil {
		t.Fatal(err)
	}

	AppVersion(writer, req)

	if writer.Code != 200 {
		t.Fatalf("Expected a 200 got %v", writer.Code)
	}

	var resp struct {
		Version string `json:"version"`
	}
	decoder := json.NewDecoder(writer.Body)
	if err := decoder.Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp.Version != Version {
		t.Fatalf("Expected version: %s got %s", Version, resp.Version)
	}
}
