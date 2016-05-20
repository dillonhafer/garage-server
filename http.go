package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

func apiLogHandler(event string) {
	fmt.Fprintln(os.Stdout, event, "-", time.Now())
}

func AppVersion(w http.ResponseWriter, r *http.Request) {
	apiLogHandler("Version")
	var jsonResp struct {
		Text string `json:"version"`
	}
	jsonResp.Text = Version
	message, err := json.Marshal(jsonResp)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	w.Write(message)
}

func CreateDoorStatusHandler(doorStatus func(int) (string, error), statusPin int) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var jsonResp struct {
			Text string `json:"door_status"`
		}

		status, err := doorStatus(statusPin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			w.WriteHeader(422)
			jsonResp.Text = fmt.Sprintf("%s", err)
		}

		jsonResp.Text = status
		message, err := json.Marshal(jsonResp)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		w.Write(message)
	})
}

type ClientRequest struct {
	Timestamp int64 `json:"timestamp"`
}

func Relay(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("signature")
	signature, err := base64.URLEncoding.DecodeString(header)
	if err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	body := buf.Bytes()

	var jsonResp struct {
		Text string `json:"status"`
	}
	jsonResp.Text = "signal received"

	verified := VerifySignature(body, signature)
	if verified {
		// Verify time
		var clientRequest ClientRequest
		err := json.Unmarshal([]byte(body), &clientRequest)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		_, err = VerifyTime(clientRequest.Timestamp)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		// Toggle switch
		apiLogHandler("TOGGLE DOOR")
		err = ToggleSwitch(options.pinNumber, options.sleepTimeout)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			w.WriteHeader(422)
			jsonResp.Text = fmt.Sprintf("%s", err)
		}
	} else {
		w.WriteHeader(401)
		apiLogHandler(fmt.Sprintf("Invalid signature: %s", signature))
		jsonResp.Text = fmt.Sprintf("%s", "Invalid signature")
	}

	message, err := json.Marshal(jsonResp)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	w.Write(message)
}
