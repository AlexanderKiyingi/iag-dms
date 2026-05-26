package config

import (
	"os"
	"strings"
)

func ListenAddr() string {
	if p := strings.TrimSpace(os.Getenv("PORT")); p != "" {
		return normalizeListenAddr(p)
	}
	if v := strings.TrimSpace(os.Getenv("ADDR")); v != "" {
		return normalizeListenAddr(v)
	}
	return ":4010"
}

func normalizeListenAddr(addr string) string {
	if !strings.HasPrefix(addr, ":") && !strings.Contains(addr, ":") {
		return ":" + addr
	}
	return addr
}
