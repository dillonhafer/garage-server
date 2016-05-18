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
	"github.com/kardianos/osext"
	"github.com/mcuadros/go-version"
	"github.com/stianeikeland/go-rpio"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const Version = "4.2.0"

var options struct {
	http            string
	pinNumber       int
	statusPinNumber int
	cert            string
	key             string
	version         bool
}

type Assets struct {
	DownloadUrl string `json:"browser_download_url"`
}

type Release struct {
	Version string   `json:"name"`
	Assets  []Assets `json:"assets"`
}

var sharedSecret = os.Getenv("GARAGE_SECRET")

func logHandler(event string) {
	fmt.Fprintln(os.Stdout, event, "-", time.Now())
}

func latestRelease() Release {
	url := "https://api.github.com/repos/dillonhafer/garage-server/releases/latest"

	res, _ := http.Get(url)
	defer res.Body.Close()

	decoder := json.NewDecoder(res.Body)
	var release Release
	var _ = decoder.Decode(&release)

	return release
}

func downloadNewRelease(url string) {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]

	dir, err := ioutil.TempDir("", "garage-server")
	if err != nil {
		fmt.Println("Error while creating tmp file", fileName, "-", err)
	}
	fileName = filepath.Join(dir, fileName)

	fmt.Println("Downloading", url, "to", fileName)

	output, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error while creating", fileName, "-", err)
		return
	}
	defer output.Close()

	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return
	}
	defer response.Body.Close()

	_, err = io.Copy(output, response.Body)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return
	}

	fmt.Println("Download finished.")

	replaceBinary(fileName)
}

func replaceBinary(path string) {
	fmt.Println("Updating server...")
	filename, _ := osext.Executable()
	fmt.Println("Copying", path, "to", filename)
	err := os.Rename(path, filename)

	if err != nil {
		fmt.Println("Could not copy file:", err)
		return
	}
}

func checkForUpdates() {
	println("Checking for updates...")
	release := latestRelease()
	fmt.Fprintf(os.Stderr, "Current version is: %s - latest version is: %s\n", Version, release.Version)

	if version.Compare(release.Version, Version, ">") {
		downloadNewRelease(release.Assets[0].DownloadUrl)
	} else {
		println("You're up to date!")
	}
}

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
		return err
	}
	pin := rpio.Pin(pinNumber)
	pin.Output()

	pin.Low()
	rpio.Close()

	time.Sleep(100 * time.Millisecond)

	err = rpio.Open()
	if err != nil {
		return err
	}
	pin.High()
	rpio.Close()

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

	verified := verifySignature(body, signature)
	if verified {
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

		// Toggle switch
		logHandler("TOGGLE DOOR")
		err = toggleSwitch(options.pinNumber)
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

	if os.Args[1] == "update" {
		checkForUpdates()
		os.Exit(0)
	}

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
