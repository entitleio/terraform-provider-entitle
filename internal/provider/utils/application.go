package utils

import "strings"

func IsApplicationWithSyncedResources(appName string) bool {
	switch strings.ToLower(appName) {
	case "manual", "virtual":
		return false
	default:
		return true
	}
}
