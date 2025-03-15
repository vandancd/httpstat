# httpstat
A utility similar to `cURL` that allows you to view http performance of a `URL`.

# Usage
`httpstat <url>`

## Helper Flags
```
-dns-servers string
  Comma-separated list of DNS server IP addresses (e.g., 8.8.8.8,8.8.4.4)
-http1
  Use HTTP/1.0
-http1.1
  Use HTTP/1.1
-ipv6
  Prefer IPv6 connections over IPv4
-max-redirects int
  Maximum number of redirects allowed (default: 5, range: 2-10)
-no-keepalive
   Disable keep-alive connections
-timeout int
  Timeout in seconds (default: 60)
```
