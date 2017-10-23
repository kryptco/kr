package kr

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const TRACKING_ID = "UA-86173430-2"

type Analytics struct{}

func (Analytics) post(clientID string, params url.Values) {
	if clientID == "disabled" {
		return
	}
	defaultParams := url.Values{
		"v":   []string{"1"},
		"tid": []string{TRACKING_ID},
		"cid": []string{clientID},
		"ua":  []string{analytics_user_agent},
		"cd1": []string{CURRENT_VERSION.String()},
		"cd2": []string{analytics_os},
		"cd7": []string{clientID},
	}
	if osVersion := getAnalyticsOSVersion(); osVersion != nil {
		defaultParams["cd3"] = []string{*osVersion}
	}
	for k, v := range params {
		defaultParams[k] = v
	}
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	_, _ = client.PostForm("https://www.google-analytics.com/collect", defaultParams)
}

func (Analytics) PostEvent(clientID string, category string, action string, label *string, value *uint64) {
	params := url.Values{
		"t":  []string{"event"},
		"ec": []string{category},
		"ea": []string{action},
	}
	if label != nil {
		params["el"] = []string{*label}
	}
	if value != nil {
		params["ev"] = []string{strconv.FormatUint(*value, 10)}
	}
	Analytics{}.post(clientID, params)
}

func readAnalyticsIDFromPersistedPairing() (id string, err error) {
	krdir, err := KrDir()
	if err != nil {
		return
	}
	persister := FilePersister{
		PairingDir: krdir,
	}
	pairing, err := persister.LoadPairing()
	if err != nil {
		return
	}
	if pairing.trackingID == nil {
		err = errors.New("no tracking ID")
		return
	}
	id = *pairing.trackingID
	return
}

func (a Analytics) PostEventUsingPersistedTrackingID(category string, action string, label *string, value *uint64) {
	id, err := readAnalyticsIDFromPersistedPairing()
	if err != nil {
		return
	}
	a.PostEvent(id, category, action, label, value)
}

func (r Request) AnalyticsTag() *string {
	if r.GitSignRequest != nil {
		tag := r.GitSignRequest.AnalyticsTag()
		return &tag
	}
	if r.SignRequest != nil {
		tag := "signature"
		return &tag
	}
	return nil
}

func (gsr GitSignRequest) AnalyticsTag() string {
	if gsr.Commit != nil {
		return "git-commit-signature"
	} else {
		return "git-tag-signature"
	}
}
