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

func DummyLogger(s string) {}

func TestVersion(t *testing.T) {
	writer := httptest.NewRecorder()
	validTimestamp := CreateTimestamp(0)

	req, err := http.NewRequest("GET", "/version", nil)
	req.Header.Add("signature", CreateSignature([]byte(validTimestamp), SharedSecret))
	req.Header.Add("timestamp", validTimestamp)
	if err != nil {
		t.Fatal(err)
	}

	AppVersion := AuthenticatedHandler(VersionHandler(DummyLogger))
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

func TestUnverifiedSignatureOnVersion(t *testing.T) {
	writer := httptest.NewRecorder()
	validTimestamp := CreateTimestamp(0)

	req, err := http.NewRequest("GET", "/version", nil)
	req.Header.Add("signature", CreateSignature([]byte(validTimestamp), "Unverified Signature"))
	req.Header.Add("timestamp", validTimestamp)
	if err != nil {
		t.Fatal(err)
	}

	AppVersion := AuthenticatedHandler(VersionHandler(DummyLogger))
	AppVersion(writer, req)
	responseEqual(t, writer.Code, 403)
}

func TestExpiredTimestampOnVersion(t *testing.T) {
	writer := httptest.NewRecorder()
	expiredTimestamp := CreateTimestamp(20)

	req, err := http.NewRequest("GET", "/version", nil)
	req.Header.Add("signature", CreateSignature([]byte(expiredTimestamp), SharedSecret))
	req.Header.Add("timestamp", expiredTimestamp)
	if err != nil {
		t.Fatal(err)
	}

	AppVersion := AuthenticatedHandler(VersionHandler(DummyLogger))
	AppVersion(writer, req)
	responseEqual(t, writer.Code, 403)
}

func TestOpenOnStatus(t *testing.T) {
	writer := httptest.NewRecorder()
	validTimestamp := CreateTimestamp(0)

	req, err := http.NewRequest("GET", "/status", nil)
	req.Header.Add("signature", CreateSignature([]byte(validTimestamp), SharedSecret))
	req.Header.Add("timestamp", validTimestamp)
	if err != nil {
		t.Fatal(err)
	}

	Status := AuthenticatedHandler(DoorStatusHandler(CreateDummyStatus("open"), DummyLogger, 0))
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

func TestClosedOnStatus(t *testing.T) {
	writer := httptest.NewRecorder()
	validTimestamp := CreateTimestamp(0)

	req, err := http.NewRequest("GET", "/status", nil)
	req.Header.Add("signature", CreateSignature([]byte(validTimestamp), SharedSecret))
	req.Header.Add("timestamp", validTimestamp)
	if err != nil {
		t.Fatal(err)
	}

	Status := AuthenticatedHandler(DoorStatusHandler(CreateDummyStatus("closed"), DummyLogger, 0))
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

func TestErrorOnStatus(t *testing.T) {
	writer := httptest.NewRecorder()
	validTimestamp := CreateTimestamp(0)

	req, err := http.NewRequest("GET", "/status", nil)
	req.Header.Add("signature", CreateSignature([]byte(validTimestamp), SharedSecret))
	req.Header.Add("timestamp", validTimestamp)
	if err != nil {
		t.Fatal(err)
	}

	Status := AuthenticatedHandler(DoorStatusHandler(CreateDummyStatus("error"), DummyLogger, 0))
	Status(writer, req)

	responseEqual(t, writer.Code, 422)
}

func TestExpiredTimestampOnStatus(t *testing.T) {
	writer := httptest.NewRecorder()
	expiredTimestamp := CreateTimestamp(20)

	req, err := http.NewRequest("GET", "/status", nil)
	req.Header.Add("signature", CreateSignature([]byte(expiredTimestamp), SharedSecret))
	req.Header.Add("timestamp", expiredTimestamp)
	if err != nil {
		t.Fatal(err)
	}

	Status := AuthenticatedHandler(DoorStatusHandler(CreateDummyStatus("open"), DummyLogger, 0))
	Status(writer, req)
	responseEqual(t, writer.Code, 403)
}

func TestUnverifiedSignatureOnStatus(t *testing.T) {
	writer := httptest.NewRecorder()
	validTimestamp := CreateTimestamp(0)

	req, err := http.NewRequest("GET", "/status", nil)
	req.Header.Add("signature", CreateSignature([]byte(validTimestamp), "Unverified Signature"))
	req.Header.Add("timestamp", validTimestamp)
	if err != nil {
		t.Fatal(err)
	}

	Status := AuthenticatedHandler(DoorStatusHandler(CreateDummyStatus("error"), DummyLogger, 0))
	Status(writer, req)

	responseEqual(t, writer.Code, 403)
}

func TestSuccessfulToggleRelay(t *testing.T) {
	writer := httptest.NewRecorder()
	validTimestamp := CreateTimestamp(0)

	req, err := http.NewRequest("GET", "/toggle", nil)
	req.Header.Add("signature", CreateSignature([]byte(validTimestamp), SharedSecret))
	req.Header.Add("timestamp", validTimestamp)
	if err != nil {
		t.Fatal(err)
	}

	Relay := AuthenticatedHandler(RelayHandle(CreateDummyRelay(false), DummyLogger, 0, 1))
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

func TestUnverifiedSignatureOnRelay(t *testing.T) {
	writer := httptest.NewRecorder()
	timestamp := CreateTimestamp(0)

	req, err := http.NewRequest("GET", "/toggle", nil)
	req.Header.Add("signature", CreateSignature([]byte(timestamp), "Unverified Signature"))
	req.Header.Add("timestamp", timestamp)
	if err != nil {
		t.Fatal(err)
	}

	Relay := AuthenticatedHandler(RelayHandle(CreateDummyRelay(false), DummyLogger, 0, 1))
	Relay(writer, req)
	responseEqual(t, writer.Code, 403)
}

func TestExpiredTimestampOnRelay(t *testing.T) {
	writer := httptest.NewRecorder()
	expiredTimestamp := CreateTimestamp(20)

	req, err := http.NewRequest("GET", "/toggle", nil)
	req.Header.Add("signature", CreateSignature([]byte(expiredTimestamp), SharedSecret))
	req.Header.Add("timestamp", expiredTimestamp)
	if err != nil {
		t.Fatal(err)
	}

	Relay := AuthenticatedHandler(RelayHandle(CreateDummyRelay(false), DummyLogger, 0, 1))
	Relay(writer, req)

	responseEqual(t, writer.Code, 403)
}

func TestToggleFailedRelay(t *testing.T) {
	writer := httptest.NewRecorder()
	validTimestamp := CreateTimestamp(0)

	req, err := http.NewRequest("GET", "/toggle", nil)
	req.Header.Add("signature", CreateSignature([]byte(validTimestamp), SharedSecret))
	req.Header.Add("timestamp", validTimestamp)

	if err != nil {
		t.Fatal(err)
	}

	Relay := AuthenticatedHandler(RelayHandle(CreateDummyRelay(true), DummyLogger, 0, 1))
	Relay(writer, req)

	responseEqual(t, writer.Code, 500)
}
