package kr

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

var analytics_user_agent = fmt.Sprintf("Mozilla/5.0 (X11; Linux) (KHTML, like Gecko) Version/%s kr/%s", CURRENT_VERSION, CURRENT_VERSION)

const analytics_os = "Linux"

var cachedAnalyticsOSVersion *string
var osVersionMutex sync.Mutex

func getAnalyticsOSVersion() *string {
	osVersionMutex.Lock()
	defer osVersionMutex.Unlock()
	if cachedAnalyticsOSVersion != nil {
		return cachedAnalyticsOSVersion
	}

	analytics_os_version_bytes, err := exec.Command("lsb_release", "-d", "-s").Output()
	if err != nil {
		log.Error("error retrieving OS version:", err.Error())
		return nil
	}
	stripped := strings.TrimSpace(string(analytics_os_version_bytes))
	cachedAnalyticsOSVersion = &stripped
	return cachedAnalyticsOSVersion
}
