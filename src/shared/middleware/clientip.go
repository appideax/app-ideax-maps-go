package middleware

import (
	"net"
	"strings"

	"github.com/gofiber/fiber/v2"
)

var trustedProxyNets []*net.IPNet

func SetTrustedProxies(entries []string) {
	nets := make([]*net.IPNet, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if !strings.Contains(entry, "/") {
			if strings.Contains(entry, ":") {
				entry += "/128"
			} else {
				entry += "/32"
			}
		}
		if _, ipNet, err := net.ParseCIDR(entry); err == nil {
			nets = append(nets, ipNet)
		}
	}
	trustedProxyNets = nets
}

func ClientIP(c *fiber.Ctx) string {
	if ip := clientFromChain(c.IP()); ip != "" {
		return ip
	}
	if remote := c.Context().RemoteIP(); remote != nil {
		return remote.String()
	}
	return ""
}

func clientFromChain(chain string) string {
	parts := strings.Split(chain, ",")
	last := strings.TrimSpace(parts[len(parts)-1])
	for i := len(parts) - 1; i >= 0; i-- {
		candidate := strings.TrimSpace(parts[i])
		if candidate == "" {
			continue
		}
		ip := net.ParseIP(candidate)
		if ip == nil {
			return last
		}
		if !isTrustedProxy(ip) {
			return candidate
		}
	}
	return last
}

func isTrustedProxy(ip net.IP) bool {
	for _, ipNet := range trustedProxyNets {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}
