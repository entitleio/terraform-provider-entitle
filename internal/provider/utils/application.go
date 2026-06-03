package utils

import "strings"

func IsApplicationWithSyncedResources(appName string) bool {
	switch strings.ToLower(appName) {
	case "manual", "virtual", "virtual application":
		return false
	default:
		return true
	}
}
