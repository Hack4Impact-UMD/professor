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
	"PLAYWRIGHT_",
}

func isSensitiveEnvKey(key string) bool {
	upper := strings.ToUpper(key)
	return strings.Contains(upper, "TOKEN") ||
		strings.Contains(upper, "SECRET") ||
		strings.Contains(upper, "PASSWORD") ||
		strings.Contains(upper, "AUTH") ||
		strings.Contains(upper, "KEY")
}

func SandboxedEnv() []string {
	var filtered []string
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		if isSensitiveEnvKey(key) {
			continue
		}

		for _, prefix := range safeEnvPrefixes {
			if strings.HasPrefix(kv, prefix) {
				filtered = append(filtered, kv)
				break
			}
		}
	}
	return filtered
}
