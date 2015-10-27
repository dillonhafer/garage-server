package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/stianeikeland/go-rpio"
	"net/http"
	"os"
	"strconv"
	"time"
)

const Version = "2.0.0"

var options struct {
	http      string
	pinNumber int
	cert      string
	key       string
	version   bool
}

var sharedSecret = os.Getenv("GARAGE_SECRET")

func verifySignature(timestamp int, signature []byte) bool {
	mac := hmac.New(sha512.New, []byte(sharedSecret))
	mac.Write([]byte(strconv.Itoa(timestamp)))
	expectedMAC := []byte(hex.EncodeToString(mac.Sum(nil)))
	return hmac.Equal(signature, expectedMAC)
}

func verifyTime(timestamp int) (int, error) {
	time64 := int64(timestamp)
	timeSinceRequest := time.Now().Unix() - time64
	if timeSinceRequest > 10 {
		return -1, errors.New("Timestamp is too far in the past")
	}
	return timestamp, nil
}

func toggleSwitch(pinNumber int) (err error) {
	err = rpio.Open()
	if err != nil {
		return
	}
	pin := rpio.Pin(pinNumber)
	pin.Output()
	pin.Low()
	time.Sleep(500 * time.Millisecond)
	pin.High()
	return nil
}

type AppRequest struct {
	Timestamp int    `json:"timestamp"`
	Signature []byte `json:"signature"`
}

func Relay(w http.ResponseWriter, r *http.Request) {
	var jsonResp struct {
		Text string `json:"status"`
	}
	jsonResp.Text = "signal received"

	decoder := json.NewDecoder(r.Body)
	var appRequest AppRequest
	err := decoder.Decode(&appRequest)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	verified := verifySignature(appRequest.Timestamp, appRequest.Signature)
	if verified {
		fmt.Println("Signature verified")
		_, err := verifyTime(appRequest.Timestamp)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Println("Time verified")
		err = toggleSwitch(options.pinNumber)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			w.WriteHeader(422)
			jsonResp.Text = fmt.Sprintf("%s", err)
		}
	} else {
		w.WriteHeader(401)
		jsonResp.Text = fmt.Sprintf("%s", "Invalid signature")
	}

	message, err := json.Marshal(jsonResp)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	w.Write(message)
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage:  %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.IntVar(&options.pinNumber, "pin", 25, "GPIO pin of relay")
	flag.StringVar(&options.http, "http", "", "HTTP listen address (e.g. 127.0.0.1:8225)")
	flag.StringVar(&options.cert, "cert", "", "SSL certificate path (e.g. /ssl/example.com.cert)")
	flag.StringVar(&options.key, "key", "", "SSL certificate key (e.g. /ssl/example.com.key)")
	flag.BoolVar(&options.version, "version", false, "print version and exit")
	flag.Parse()

	if options.version {
		fmt.Printf("garage-server v%v\n", Version)
		os.Exit(0)
	}

	if sharedSecret == "" {
		println("You did not set GARAGE_SECRET env var")
		os.Exit(1)
	}

	serveAddress := "127.0.0.1:8225"
	if options.http != "" {
		serveAddress = options.http
	}

	http.HandleFunc("/", Relay)

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
