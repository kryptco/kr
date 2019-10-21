package analytics

import (
	"fmt"
	. "krypt.co/kr/common/version"
	"sync"
)

// TODO
var analytics_user_agent = fmt.Sprintf("Mozilla/5.0 (Windows NT 0.0; Win64; x64; rv: 0.0) (KHTML, like Gecko) Version/%s kr/%s", CURRENT_VERSION, CURRENT_VERSION)

const analytics_os = "Linux"

var cachedAnalyticsOSVersion *string
var osVersionMutex sync.Mutex

func getAnalyticsOSVersion() *string {
	osVersionMutex.Lock()
	defer osVersionMutex.Unlock()
	if cachedAnalyticsOSVersion != nil {
		return cachedAnalyticsOSVersion
	}

	// TODO
	return nil
}
