package lkvs

import (
	"encoding/base64"
	"strings"
)

// FindSubDomain find sub domain
func FindSubDomain(query, zoneName string) string {
	if query == zoneName {
		return "@"
	}

	subDomain := strings.TrimSuffix(query, "."+zoneName)
	return subDomain
}

// GenerateID generate id for recored
func GenerateID(s string) string {
	b := []byte(s)

	sEnc := base64.StdEncoding.EncodeToString(b)
	return sEnc
}
