// Package hookurl validates owner-selected webhook destinations. Private and
// loopback services are supported; link-local/metadata destinations are not.
package hookurl

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
)

func AllowedIP(ip net.IP) bool {
	return ip != nil && !ip.IsUnspecified() && !ip.IsMulticast() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast()
}

func Validate(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || len(raw) > 2048 || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Fragment != "" || strings.ContainsAny(raw, "\r\n\t") {
		return errors.New("Use an HTTP or HTTPS URL without a username, password or fragment.")
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if host == "metadata.google.internal" || strings.Contains(host, "%") {
		return errors.New("Metadata and link-local destinations are not supported.")
	}
	if ip := net.ParseIP(host); ip != nil && !AllowedIP(ip) {
		return errors.New("Metadata and link-local destinations are not supported.")
	}
	if p := u.Port(); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			return errors.New("Use a port between 1 and 65535.")
		}
	}
	return nil
}
