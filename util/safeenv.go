package util

import (
	"os"
	"strings"
)

var safeEnvPrefixes = []string{
	"PATH=",
	"HOME=",
	"TMPDIR=",
	"TEMP=",
	"TMP=",
	"USER=",
	"LANG=",
	"LC_",
	"NODE_",
	"npm_",
	"PLAYWRIGHT_",
}

func SandboxedEnv() []string {
	var filtered []string
	for _, kv := range os.Environ() {
		for _, prefix := range safeEnvPrefixes {
			if strings.HasPrefix(kv, prefix) {
				filtered = append(filtered, kv)
				break
			}
		}
	}
	return filtered
}
