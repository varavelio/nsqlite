package validate

import (
	"regexp"
)

var listenHostRe = regexp.MustCompile(`^([0-9]{1,3}\.){3}[0-9]{1,3}($|/[0-9]{2})$`)

// ListenHost validates if addr is a valid host to listen on.
func ListenHost(addr string) bool {
	return listenHostRe.MatchString(addr)
}
