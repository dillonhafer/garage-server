package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

func apiLogHandler(event string) {
	fmt.Fprintln(os.Stderr, event, "-", time.Now())
}

func VersionHandler(logger func(string)) http.HandlerFunc {
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

func DoorStatusHandler(doorStatus func(int) (string, error), logger func(string), statusPin int) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var jsonResp struct {
			Text string `json:"doorStatus"`
		}

		status, err := doorStatus(statusPin)
		if err != nil {
			errMessage := fmt.Sprintf("Could not read pin '%d' on Raspberry Pi", statusPin)
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

func RelayHandle(toggleSwitch func(int, int) error, logger func(string), pinNumber int, sleepTimeout int) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger("TOGGLE DOOR")
		err := toggleSwitch(pinNumber, sleepTimeout)
		if err != nil {
			errMessage := "Could not write to pin"
			logger(errMessage)
			w.WriteHeader(500)
			return
		}

		var resp struct {
			Status string `json:"status"`
		}
		resp.Status = "signal received"
		message, err := json.Marshal(resp)
		if err != nil {
			logger(fmt.Sprintf("%s", err))
		}
		w.Write(message)
	})
}

func LogsHandler(logger func(string), logFile string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		logger("Logs")
		logs := ParseLogs(logFile)
		entries, err := json.Marshal(logs)
		if err != nil {
			logger(fmt.Sprintf("%s", err))
		}
		w.Write(entries)
	})
}

func AuthenticatedHandler(f http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		signature := req.Header.Get("signature")
		timestamp := req.Header.Get("timestamp")
		decodedSignature, err := base64.URLEncoding.DecodeString(signature)
		if err != nil {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		verified := VerifySignature([]byte(timestamp), decodedSignature)
		if verified {
			// Verify time
			i, err := strconv.ParseInt(timestamp, 10, 64)
			if err != nil {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			_, err = VerifyTime(i)
			if err != nil {
				w.WriteHeader(http.StatusForbidden)
				return
			}
		} else {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		f(w, req)
	})
}
