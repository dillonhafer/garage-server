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
	fmt.Fprintln(os.Stderr, event, "-", time.Now())
}
func CreateVersionHandler(logger func(string)) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		logger("Version")
		var jsonResp struct {
			Text string `json:"version"`
		}
		jsonResp.Text = Version
		message, err := json.Marshal(jsonResp)
		if err != nil {
			logger(fmt.Sprintf("%s", err))
		}
		w.Write(message)
	})
}

func CreateDoorStatusHandler(doorStatus func(int) (string, error), logger func(string), statusPin int) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var jsonResp struct {
			Text string `json:"door_status"`
		}

		status, err := doorStatus(statusPin)
		if err != nil {
			errMessage := fmt.Sprintf("%s", err)
			logger(errMessage)
			jsonResp.Text = errMessage
			w.WriteHeader(422)
		}

		jsonResp.Text = status
		message, err := json.Marshal(jsonResp)
		if err != nil {
			logger(fmt.Sprintf("%s", err))
		}
		w.Write(message)
	})
}

type ClientRequest struct {
	Timestamp int64 `json:"timestamp"`
}

func CreateRelayHandler(toggleSwitch func(int, int) error, logger func(string), pinNumber int, sleepTimeout int) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var jsonResp struct {
			Status string `json:"status"`
		}
		jsonResp.Status = "signal received"

		header := r.Header.Get("signature")
		signature, err := base64.URLEncoding.DecodeString(header)
		if err != nil {
			w.WriteHeader(422)
			jsonResp.Status = fmt.Sprintf("%s", err)
			return
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		body := buf.Bytes()

		verified := VerifySignature(body, signature)
		if verified {
			// Verify time
			var clientRequest ClientRequest
			err := json.Unmarshal([]byte(body), &clientRequest)
			if err != nil {
				errMessage := fmt.Sprintf("%s", err)
				logger(errMessage)
				jsonResp.Status = errMessage
				w.WriteHeader(422)
			}

			_, err = VerifyTime(clientRequest.Timestamp)
			if err != nil {
				errMessage := fmt.Sprintf("%s", err)
				logger(errMessage)
				jsonResp.Status = errMessage
				w.WriteHeader(422)
			}

			// Toggle switch
			logger("TOGGLE DOOR")
			err = toggleSwitch(pinNumber, sleepTimeout)
			if err != nil {
				errMessage := fmt.Sprintf("%s", err)
				logger(errMessage)
				jsonResp.Status = errMessage
				w.WriteHeader(500)
			}
		} else {
			logger(fmt.Sprintf("Invalid signature: %s", signature))
			jsonResp.Status = "Invalid signature"
			w.WriteHeader(401)
		}

		message, err := json.Marshal(jsonResp)
		if err != nil {
			logger(fmt.Sprintf("%s", err))
		}
		w.Write(message)
	})
}
