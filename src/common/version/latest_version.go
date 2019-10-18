package version

import (
	"encoding/json"
	"io/ioutil"
	"krypt.co/kr/common/log"
	"net/http"
	"time"

	"github.com/blang/semver"
	"github.com/op/go-logging"
	"github.com/youtube/vitess/go/ioutil2"

	. "krypt.co/kr/common/socket"
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
		log.Log.Error("Error finding home directory:", fileErr.Error())
		return
	}
	if writeErr := ioutil2.WriteFileAtomic(file, versionsJson, 0700); writeErr != nil {
		log.Log.Error("Error writing update log file:", writeErr.Error())
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

func CheckedForUpdateRecently(log *logging.Logger) bool {
	file, fileErr := KrDirFile("last_update_check")
	if fileErr != nil {
		log.Error("Error finding home directory:", fileErr.Error())
	} else {
		lastUpdateUnixSecondsBytes, readErr := ioutil.ReadFile(file)
		if readErr == nil {
			var lastUpdateUnixSeconds int64
			parseErr := json.Unmarshal(lastUpdateUnixSecondsBytes, &lastUpdateUnixSeconds)
			if parseErr == nil && (time.Now().Unix()-lastUpdateUnixSeconds) < int64(3600*5) {
				return true
			}
		}
		nowUnixSecondsBytes, marshalErr := json.Marshal(time.Now().Unix())
		if marshalErr != nil {
			log.Error("Error serializing current time:", marshalErr.Error())
		} else {
			if writeErr := ioutil2.WriteFileAtomic(file, nowUnixSecondsBytes, 0700); writeErr != nil {
				log.Error("Error writing update log file:", writeErr.Error())
			}
		}
	}
	return false
}

func CheckIfUpdateAvailable(log *logging.Logger) bool {
	var latest semver.Version
	var verErr error
	if CheckedForUpdateRecently(log) {
		log.Notice("Checked for update recently, falling back to latest version cache.")
		var cacheErr error
		latest, cacheErr = GetCachedLatestVersion()
		if cacheErr != nil {
			return false
		}
	} else {
		latest, verErr = GetLatestVersion()
	}
	if verErr == nil {
		if CURRENT_VERSION.LT(latest) {
			return true
		}
	}
	return false
}
