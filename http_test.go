package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func CreateDummyStatus(state string) func(int) (string, error) {
	return func(i int) (s string, e error) {
		var err error
		if state == "error" {
			err = errors.New("unprocessable entity")
		}
		return state, err
	}
}

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

func TestOpenStatus(t *testing.T) {
	writer := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/status", nil)
	if err != nil {
		t.Fatal(err)
	}

	Status := CreateDoorStatusHandler(CreateDummyStatus("open"), 0)
	Status(writer, req)

	if writer.Code != 200 {
		t.Fatalf("Expected a 200 got %v", writer.Code)
	}

	var resp struct {
		Status string `json:"door_status"`
	}
	decoder := json.NewDecoder(writer.Body)
	if err := decoder.Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp.Status != "open" {
		t.Fatalf("Expected status: %s got %s", "open", resp.Status)
	}
}

func TestClosedStatus(t *testing.T) {
	writer := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/status", nil)
	if err != nil {
		t.Fatal(err)
	}

	Status := CreateDoorStatusHandler(CreateDummyStatus("closed"), 0)
	Status(writer, req)

	if writer.Code != 200 {
		t.Fatalf("Expected a 200 got %v", writer.Code)
	}

	var resp struct {
		Status string `json:"door_status"`
	}
	decoder := json.NewDecoder(writer.Body)
	if err := decoder.Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp.Status != "closed" {
		t.Fatalf("Expected status: %s got %s", "closed", resp.Status)
	}
}

func TestErrorStatus(t *testing.T) {
	writer := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/status", nil)
	if err != nil {
		t.Fatal(err)
	}

	Status := CreateDoorStatusHandler(CreateDummyStatus("error"), 0)
	Status(writer, req)

	if writer.Code != 422 {
		t.Fatalf("Expected a 422 got %v", writer.Code)
	}
}
