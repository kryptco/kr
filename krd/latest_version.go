package krd

import (
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/blang/semver"
	"github.com/kryptco/kr"
	"github.com/op/go-logging"
	"github.com/youtube/vitess/go/ioutil2"
)

func CheckedForUpdateRecently(log *logging.Logger) bool {
	file, fileErr := kr.KrDirFile("last_update_check")
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
		latest, cacheErr = kr.GetCachedLatestVersion()
		if cacheErr != nil {
			return false
		}
	} else {
		latest, verErr = kr.GetLatestVersion()
	}
	if verErr == nil {
		if kr.CURRENT_VERSION.LT(latest) {
			return true
		}
	}
	return false
}
