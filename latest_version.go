package kr

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/youtube/vitess/go/ioutil2"
)

var VERSIONS_S3_BUCKET = "https://s3.amazonaws.com/kr-versions/versions"

type Versions struct {
	IOS   string `json:"iOS"`
	OSX   string `json:"osx"`
	Linux string `json:"linux"`
}

func GetLatestVersions() (versions Versions, err error) {
	httpClient := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := httpClient.Get(VERSIONS_S3_BUCKET)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	versionsJson, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(versionsJson, &versions)
	if err != nil {
		return
	}
	cacheLatestVersions(versionsJson)
	return
}

func cacheLatestVersions(versionsJson []byte) {
	file, fileErr := KrDirFile("latest_versions_cache")
	if fileErr != nil {
		log.Error("Error finding home directory:", fileErr.Error())
		return
	}
	if writeErr := ioutil2.WriteFileAtomic(file, versionsJson, 0700); writeErr != nil {
		log.Error("Error writing update log file:", writeErr.Error())
	}
}

func GetCachedLatestVersions() (versions Versions, err error) {
	cacheFile, err := KrDirFile("latest_versions_cache")
	if err != nil {
		return
	}
	cacheBytes, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		return
	}
	err = json.Unmarshal(cacheBytes, &versions)
	return
}
