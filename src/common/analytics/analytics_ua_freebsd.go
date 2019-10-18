package analytics

import (
	"fmt"
	"krypt.co/kr/common/log"
	"krypt.co/kr/common/version"
	"os/exec"
	"strings"
	"sync"
)

var analytics_user_agent = fmt.Sprintf("Mozilla/5.0 (X11; FreeBSD) (KHTML, like Gecko) Version/%s kr/%s", version.CURRENT_VERSION, version.CURRENT_VERSION)

const analytics_os = "FreeBSD"

var cachedAnalyticsOSVersion *string
var osVersionMutex sync.Mutex

func getAnalyticsOSVersion() *string {
	osVersionMutex.Lock()
	defer osVersionMutex.Unlock()
	if cachedAnalyticsOSVersion != nil {
		return cachedAnalyticsOSVersion
	}

	analytics_os_version_bytes, err := exec.Command("freebsd-version").Output()
	if err != nil {
		log.Log.Error("error retrieving OS version:", err.Error())
		return nil
	}
	stripped := strings.TrimSpace(string(analytics_os_version_bytes))
	cachedAnalyticsOSVersion = &stripped
	return cachedAnalyticsOSVersion
}
