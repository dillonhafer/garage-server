package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

const Version = "4.4.0"

var options struct {
	http            string
	pinNumber       int
	statusPinNumber int
	sleepTimeout    int
	cert            string
	key             string
	version         bool
}

var SharedSecret = os.Getenv("GARAGE_SECRET")

func logHandler(event string) {
	fmt.Fprintln(os.Stdout, event, "-", time.Now())
}

type ClientRequest struct {
	Timestamp int64 `json:"timestamp"`
}

func Status(w http.ResponseWriter, r *http.Request) {
	var jsonResp struct {
		Text string `json:"door_status"`
	}

	status, err := CheckDoorStatus(options.statusPinNumber)
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
}

func AppVersion(w http.ResponseWriter, r *http.Request) {
	logHandler("Version")
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
		logHandler("TOGGLE DOOR")
		err = ToggleSwitch(options.pinNumber, options.sleepTimeout)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			w.WriteHeader(422)
			jsonResp.Text = fmt.Sprintf("%s", err)
		}
	} else {
		w.WriteHeader(401)
		logHandler(fmt.Sprintf("Invalid signature: %s", signature))
		jsonResp.Text = fmt.Sprintf("%s", "Invalid signature")
	}

	message, err := json.Marshal(jsonResp)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	w.Write(message)
}

func main() {
	if os.Args[1] == "update" {
		CheckForUpdates()
		os.Exit(0)
	}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage:  %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.IntVar(&options.pinNumber, "pin", 25, "GPIO pin of relay")
	flag.IntVar(&options.statusPinNumber, "status-pin", 10, "GPIO pin of reed switch")
	flag.IntVar(&options.sleepTimeout, "sleep", 100, "Time in milliseconds to keep switch closed")
	flag.StringVar(&options.http, "http", "", "HTTP listen address (e.g. 127.0.0.1:8225)")
	flag.StringVar(&options.cert, "cert", "", "SSL certificate path (e.g. /ssl/example.com.cert)")
	flag.StringVar(&options.key, "key", "", "SSL certificate key (e.g. /ssl/example.com.key)")
	flag.BoolVar(&options.version, "version", false, "print version and exit")
	flag.Parse()

	if options.version {
		fmt.Printf("garage-server v%v\n", Version)
		os.Exit(0)
	}

	if SharedSecret == "" {
		println("You did not set GARAGE_SECRET env var")
		os.Exit(1)
	}

	serveAddress := "127.0.0.1:8225"
	if options.http != "" {
		serveAddress = options.http
	}

	http.HandleFunc("/", Relay)
	http.HandleFunc("/status", Status)
	http.HandleFunc("/version", AppVersion)

	fmt.Fprintln(os.Stderr, "=> Booting Garage Server ", Version)
	fmt.Fprintln(os.Stderr, "=> Run `garage-server -h` for more startup options")
	fmt.Fprintln(os.Stderr, "=> Ctrl-C to shutdown server")

	var err error
	if options.key != "" && options.cert != "" {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("* Listening on https://%s", serveAddress))
		err = http.ListenAndServeTLS(serveAddress, options.cert, options.key, nil)
	} else {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("* Listening on http://%s", serveAddress))
		err = http.ListenAndServe(serveAddress, nil)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
