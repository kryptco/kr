package krdclient

import (
	"os"
	"strings"
"fmt"
)

//	parse current command if dynamically loaded with ssh
func currentCommand() *string {
	cmd := os.Getenv("$0")
fmt.Println(cmd)
	toks := strings.Fields(cmd)
	if cmd == "" || len(toks) == 0 {
		return nil
	}
	if !strings.HasSuffix(toks[0], "ssh") {
		return nil
	}
	return &cmd
}
