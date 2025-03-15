build:
	env GOOS=linux GOARCH=amd64 go build -ldflags '-s -w' -o httpstat  github.com/vandancd/httpstat
build-mchip:
	env GOOS=darwin GOARCH=arm64 go build -ldflags '-s -w' -o httpstat github.com/vandancd/httpstat