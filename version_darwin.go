package kr

import (
	"github.com/blang/semver"
)

var CURRENT_VERSION = semver.MustParse("2.3.0")

func GetLatestVersion() (version semver.Version, err error) {
	versions, err := GetLatestVersions()
	if err != nil {
		return
	}
	version, err = semver.Make(versions.OSX)
	return
}

func GetCachedLatestVersion() (version semver.Version, err error) {
	versions, err := GetCachedLatestVersions()
	if err != nil {
		return
	}
	version, err = semver.Make(versions.OSX)
	return
}
