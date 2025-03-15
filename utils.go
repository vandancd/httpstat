package main

import "strings"

// normalizeURL ensures the URL has a proper scheme prefix
func normalizeURL(url string) string {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "http://" + url
	}
	return url
}

// connectionInfo returns a string indicating whether a connection was reused
func connectionInfo(reused bool) string {
	if reused {
		return "reused"
	}
	return "new"
}
