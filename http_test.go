package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func responseEqual(t *testing.T, a int, b int) {
	if a != b {
		t.Fatalf("Expected response to be %d but was %d", b, a)
	}
}

func stringEqual(t *testing.T, a string, b string) {
	if a != b {
		t.Fatalf("Expected '%s' to equal '%s'", a, b)
	}
}

func CreateDummyStatus(state string) func(int) (string, error) {
	return func(i int) (s string, e error) {
		if state == "error" {
			e = errors.New("unprocessable entity")
		}
		return state, e
	}
}

func CreateDummyRelay(bad bool) func(int, int) error {
	return func(i int, ii int) (e error) {
		if bad {
			e = errors.New("open /dev/mem: no such file or directory")
		}
		return e
	}
}

func DummyLogger(s string) {}

func TestVersion(t *testing.T) {
	writer := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/version", nil)
	if err != nil {
		t.Fatal(err)
	}

	AppVersion := CreateVersionHandler(DummyLogger)
	AppVersion(writer, req)
	responseEqual(t, writer.Code, 200)

	var resp struct {
		Version string `json:"version"`
	}
	decoder := json.NewDecoder(writer.Body)
	if err := decoder.Decode(&resp); err != nil {
		t.Fatal(err)
	}

	stringEqual(t, resp.Version, Version)
}

func TestOpenStatus(t *testing.T) {
	writer := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/status", nil)
	if err != nil {
		t.Fatal(err)
	}

	Status := CreateDoorStatusHandler(CreateDummyStatus("open"), DummyLogger, 0)
	Status(writer, req)

	responseEqual(t, writer.Code, 200)

	var resp struct {
		Status string `json:"door_status"`
	}
	decoder := json.NewDecoder(writer.Body)
	if err := decoder.Decode(&resp); err != nil {
		t.Fatal(err)
	}
	stringEqual(t, resp.Status, "open")
}

func TestClosedStatus(t *testing.T) {
	writer := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/status", nil)
	if err != nil {
		t.Fatal(err)
	}

	Status := CreateDoorStatusHandler(CreateDummyStatus("closed"), DummyLogger, 0)
	Status(writer, req)

	responseEqual(t, writer.Code, 200)

	var resp struct {
		Status string `json:"door_status"`
	}
	decoder := json.NewDecoder(writer.Body)
	if err := decoder.Decode(&resp); err != nil {
		t.Fatal(err)
	}

	stringEqual(t, resp.Status, "closed")
}

func TestErrorStatus(t *testing.T) {
	writer := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/status", nil)
	if err != nil {
		t.Fatal(err)
	}

	Status := CreateDoorStatusHandler(CreateDummyStatus("error"), DummyLogger, 0)
	Status(writer, req)

	responseEqual(t, writer.Code, 422)
}

func CreateSignature(body []byte, secret string) string {
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(body)
	expectedMAC := []byte(hex.EncodeToString(mac.Sum(nil)))
	return base64.URLEncoding.EncodeToString(expectedMAC)
}

func CreateTimestamp(offset int64) string {
	validTime := time.Now().Unix() - offset
	return fmt.Sprintf("%d", validTime)
}

func TestSuccessfulToggleRelay(t *testing.T) {
	writer := httptest.NewRecorder()
	validTimestamp := CreateTimestamp(0)

	req, err := http.NewRequest("GET", "/", nil)
	req.Header.Add("signature", CreateSignature([]byte(validTimestamp), SharedSecret))
	req.Header.Add("timestamp", validTimestamp)
	if err != nil {
		t.Fatal(err)
	}

	Relay := CreateRelayHandler(CreateDummyRelay(false), DummyLogger, 0, 1)
	Relay(writer, req)

	responseEqual(t, writer.Code, 200)

	var resp struct {
		Status string `json:"status"`
	}

	decoder := json.NewDecoder(writer.Body)
	if err := decoder.Decode(&resp); err != nil {
		t.Fatal(err)
	}

	stringEqual(t, resp.Status, "signal received")
}

func TestUnverifiedSignatureRelay(t *testing.T) {
	writer := httptest.NewRecorder()
	timestamp := CreateTimestamp(0)

	req, err := http.NewRequest("GET", "/", nil)
	req.Header.Add("signature", CreateSignature([]byte(timestamp), "Bad Signature"))
	req.Header.Add("timestamp", timestamp)
	if err != nil {
		t.Fatal(err)
	}

	Relay := CreateRelayHandler(CreateDummyRelay(false), DummyLogger, 0, 1)
	Relay(writer, req)

	responseEqual(t, writer.Code, 401)

	var resp struct {
		Status string `json:"status"`
	}

	decoder := json.NewDecoder(writer.Body)
	if err := decoder.Decode(&resp); err != nil {
		t.Fatal(err)
	}

	stringEqual(t, resp.Status, "Invalid signature")
}

func TestExpiredRequestRelay(t *testing.T) {
	writer := httptest.NewRecorder()
	invalidTimestamp := CreateTimestamp(20)

	req, err := http.NewRequest("GET", "/", nil)
	req.Header.Add("signature", CreateSignature([]byte(invalidTimestamp), SharedSecret))
	req.Header.Add("timestamp", invalidTimestamp)
	if err != nil {
		t.Fatal(err)
	}

	Relay := CreateRelayHandler(CreateDummyRelay(false), DummyLogger, 0, 1)
	Relay(writer, req)

	responseEqual(t, writer.Code, 422)

	var resp struct {
		Status string `json:"status"`
	}

	decoder := json.NewDecoder(writer.Body)
	if err := decoder.Decode(&resp); err != nil {
		t.Fatal(err)
	}

	stringEqual(t, resp.Status, "Timestamp is too far in the past")
}

func TestToggleFailedRelay(t *testing.T) {
	writer := httptest.NewRecorder()
	validTimestamp := CreateTimestamp(0)

	req, err := http.NewRequest("GET", "/", nil)
	req.Header.Add("signature", CreateSignature([]byte(validTimestamp), SharedSecret))
	req.Header.Add("timestamp", validTimestamp)

	if err != nil {
		t.Fatal(err)
	}

	Relay := CreateRelayHandler(CreateDummyRelay(true), DummyLogger, 0, 1)
	Relay(writer, req)

	responseEqual(t, writer.Code, 500)

	var resp struct {
		Status string `json:"status"`
	}

	decoder := json.NewDecoder(writer.Body)
	if err := decoder.Decode(&resp); err != nil {
		t.Fatal(err)
	}

	stringEqual(t, resp.Status, "Could not write to pin")
}
