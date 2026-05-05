package validate

import (
	"regexp"
	"strconv"
)

var portRe = regexp.MustCompile(`^\d{1,5}$`)

// Port validates if port is a valid port number.
func Port(port string) bool {
	if !portRe.MatchString(port) {
		return false
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return false
	}

	if portInt < 1 || portInt > 65535 {
		return false
	}

	return true
}
