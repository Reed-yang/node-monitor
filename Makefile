VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build clean test

build:
	go build -ldflags "$(LDFLAGS)" -o node-monitor .

test:
	go test ./... -v

clean:
	rm -f node-monitor

install: build
	cp node-monitor ~/.local/bin/node-monitor
