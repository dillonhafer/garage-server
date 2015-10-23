package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
)

const Version = "1.0.0"

var options struct {
	httpAddr  string
	pinNumber int
	version   bool
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage:  %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.IntVar(&options.pinNumber, "pin", 25, "GPIO pin of relay")
	flag.StringVar(&options.httpAddr, "http", "", "HTTP listen address (e.g. 127.0.0.1:8225)")
	flag.BoolVar(&options.version, "version", false, "print version and exit")

	flag.Parse()

	if options.version {
		fmt.Printf("garage-server v%v\n", Version)
		os.Exit(0)
	}

	serveAddress := "127.0.0.1:8225"
	if options.httpAddr != "" {
		serveAddress = options.httpAddr
	}
	fmt.Fprintln(os.Stderr, "Listening on:", serveAddress)
	fmt.Fprintln(os.Stderr, "Use `--httpAddr` flag to change the default address")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var jsonResp struct {
			Text string `json:"text"`
		}
		jsonResp.Text = "Open/Close sent"
		js, err := json.Marshal(jsonResp)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		w.Write(js)
	})

	err := http.ListenAndServe(serveAddress, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
