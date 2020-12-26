package lkvs

import (
	"encoding/base64"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
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

var logger *log.Logger

func init() {
	logger = log.New()

	logger.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
	logger.SetReportCaller(true)

	logger.SetOutput(os.Stdout)

	runMode := os.Getenv("DEBUG_ON")
	if runMode != "" {
		logger.SetLevel(log.DebugLevel)
	} else {
		logger.SetLevel(log.WarnLevel)
	}
}

func IsPublicDomain(name string) (is bool) {
	is = true
	_, err := net.LookupNS(name)
	if err != nil {
		is = false
	}
	return is
}
