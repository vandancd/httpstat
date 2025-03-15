package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
)

// getSystemDNSServers reads system DNS servers from resolv.conf
func getSystemDNSServers() []string {
	file, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return nil
	}
	defer file.Close()

	var servers []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "nameserver") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				servers = append(servers, fields[1])
			}
		}
	}
	return servers
}

// createCustomResolver creates a custom DNS resolver that tries multiple DNS servers
func createCustomResolver(dnsServers []string) *net.Resolver {
	currentServer := 0
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			var lastErr error
			for i := 0; i < len(dnsServers); i++ {
				server := dnsServers[currentServer]
				currentServer = (currentServer + 1) % len(dnsServers)

				addTraceMessage("Attempting DNS resolution using server: %s", server)
				conn, err := net.Dial("udp", server+":53")
				if err == nil {
					return conn, nil
				}
				lastErr = err
			}
			return nil, fmt.Errorf("all DNS servers failed, last error: %v", lastErr)
		},
	}
}
