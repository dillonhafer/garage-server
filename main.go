package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
)

const Version = "5.0.1"

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

func main() {
	if len(os.Args) > 1 && os.Args[1] == "update" {
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

	Relay := CreateRelayHandler(ToggleSwitch, apiLogHandler, options.pinNumber, options.sleepTimeout)
	Status := CreateDoorStatusHandler(CheckDoorStatus, apiLogHandler, options.statusPinNumber)
	AppVersion := CreateVersionHandler(apiLogHandler)

	http.HandleFunc("/toggle", Relay)
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
