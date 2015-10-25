package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/stianeikeland/go-rpio"
	"net/http"
	"os"
	"time"
)

const Version = "1.0.0"

var options struct {
	http      string
	pinNumber int
	version   bool
}

var sharedSecret = os.Getenv("GARAGE_SECRET")

func CheckMAC(message, messageMAC []byte) bool {
	mac := hmac.New(sha512.New, []byte(sharedSecret))
	mac.Write(message)
	expectedMAC := []byte(hex.EncodeToString(mac.Sum(nil)))
	return hmac.Equal(messageMAC, expectedMAC)
}

func verifyTime(decodedJSON []byte) (map[string]int, error) {
	payload := make(map[string]int)
	err := json.Unmarshal(decodedJSON, &payload)
	if err != nil {
		return nil, err
	}

	time64 := int64(payload["timestamp"])
	if (time.Now().Unix() - time64) > 30 {
		return nil, errors.New("Timestamp is too far in the past")
	}

	return payload, nil
}

func toggleSwitch(pinNumber int) (err error) {
	err = rpio.Open()
	if err != nil {
		return
	}
	pin := rpio.Pin(pinNumber)
	pin.Output()
	pin.High()
	time.Sleep(1 * time.Second)
	pin.Low()
	return nil
}

type AppRequest struct {
	Data      string `json:"data"`
	Signature string `json:"signature"`
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage:  %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.IntVar(&options.pinNumber, "pin", 25, "GPIO pin of relay")
	flag.StringVar(&options.http, "http", "", "HTTP listen address (e.g. 127.0.0.1:8225)")
	flag.BoolVar(&options.version, "version", false, "print version and exit")

	flag.Parse()

	if sharedSecret == "" {
		println("You did not set GARAGE_SECRET env var")
		os.Exit(0)
	}

	if options.version {
		fmt.Printf("garage-server v%v\n", Version)
		os.Exit(0)
	}

	serveAddress := "127.0.0.1:8225"
	if options.http != "" {
		serveAddress = options.http
	}
	fmt.Fprintln(os.Stderr, "Listening on:", serveAddress)
	fmt.Fprintln(os.Stderr, "Use `--http` flag to change the default address")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var jsonResp struct {
			Text string `json:"status"`
		}
		jsonResp.Text = "signal received"

		decoder := json.NewDecoder(r.Body)
		var appRequest AppRequest
		err := decoder.Decode(&appRequest)
		if err != nil {
			panic("Could Not Decode JSON")
		}

		signature := appRequest.Signature
		decodedSignature, err := base64.URLEncoding.DecodeString(signature)
		if err != nil {
			panic(err)
		}

		data := appRequest.Data
		decodedJSON, err := base64.URLEncoding.DecodeString(data)
		if err != nil {
			panic(err)
		}

		verified := CheckMAC(decodedJSON, decodedSignature)
		if verified {
			fmt.Println("Signature verified")
			_, err := verifyTime(decodedJSON)
			if err != nil {
				panic(err)
			}

			fmt.Println("Time verified")
			err = toggleSwitch(options.pinNumber)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				w.WriteHeader(422)
				jsonResp.Text = fmt.Sprintf("%s", err)
			}
		} else {
			w.WriteHeader(422)
			jsonResp.Text = fmt.Sprintf("%s", "Invalid signature")
		}

		message, err := json.Marshal(jsonResp)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		w.Write(message)
	})

	err := http.ListenAndServe(serveAddress, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
