package main

import (
	"bytes"
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

const Version = "3.2.0"

var options struct {
	http            string
	pinNumber       int
	statusPinNumber int
	cert            string
	key             string
	version         bool
}

var sharedSecret = os.Getenv("GARAGE_SECRET")

func verifySignature(signedText []byte, signature []byte) bool {
	mac := hmac.New(sha512.New, []byte(sharedSecret))
	mac.Write(signedText)
	expectedMAC := []byte(hex.EncodeToString(mac.Sum(nil)))
	return hmac.Equal(signature, expectedMAC)
}

func verifyTime(timestamp int64) (int64, error) {
	timeSinceRequest := time.Now().Unix() - timestamp
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
	defer rpio.Close()

	pin := rpio.Pin(pinNumber)
	pin.Output()
	pin.Low()
	time.Sleep(500 * time.Millisecond)
	pin.High()
	return nil
}

type ClientRequest struct {
	Timestamp int64 `json:"timestamp"`
}

func CheckStatus(pinNumber int) (state string, err error) {
	err = rpio.Open()
	if err != nil {
		return
	}
	defer rpio.Close()

	pin := rpio.Pin(pinNumber)

	status := "open"
	if pin.Read() == 0 {
		status = "closed"
	}

	return status, err
}

func DoorStatus(w http.ResponseWriter, r *http.Request) {
	var jsonResp struct {
		Text string `json:"door_status"`
	}

	status, err := CheckStatus(options.statusPinNumber)
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

func GetVersion(w http.ResponseWriter, r *http.Request) {
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

	verified := verifySignature(body, signature)
	if verified {
		fmt.Println("Signature verified")

		// Verify time
		var clientRequest ClientRequest
		err := json.Unmarshal([]byte(body), &clientRequest)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		_, err = verifyTime(clientRequest.Timestamp)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		fmt.Println("Time verified")

		// Toggle switch
		err = toggleSwitch(options.pinNumber)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			w.WriteHeader(422)
			jsonResp.Text = fmt.Sprintf("%s", err)
		}
	} else {
		w.WriteHeader(401)
		println(fmt.Sprintf("Invalid signature:%s", signature))
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
	flag.IntVar(&options.statusPinNumber, "status-pin", 10, "GPIO pin of reed switch")
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
	http.HandleFunc("/status", DoorStatus)
	http.HandleFunc("/version", GetVersion)

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
