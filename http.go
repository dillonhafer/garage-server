package main

import (
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
