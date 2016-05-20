package main

import (
	"encoding/json"
	"fmt"
	"github.com/kardianos/osext"
	"github.com/mcuadros/go-version"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Assets struct {
	DownloadUrl string `json:"browser_download_url"`
}

type Release struct {
	Version string   `json:"name"`
	Assets  []Assets `json:"assets"`
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

func CheckForUpdates() {
	println("Checking for updates...")
	release := latestRelease()
	fmt.Fprintf(os.Stderr, "Current version is: %s - latest version is: %s\n", Version, release.Version)

	if version.Compare(release.Version, Version, ">") {
		downloadNewRelease(release.Assets[0].DownloadUrl)
	} else {
		println("You're up to date!")
	}
}
