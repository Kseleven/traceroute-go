GOSRC = $(shell find . -type f -name '*.go')

build: trace

trace: $(GOSRC)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o traceroute-go cmd/main.go

clean:
	rm -rf traceroute-go
