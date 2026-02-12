BINARY_NAME=skill-installer
VERSION=3.0.0

.PHONY: build clean test install

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY_NAME) .

clean:
	rm -f $(BINARY_NAME)

test:
	go test ./...

install: build
	install -m 755 $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
