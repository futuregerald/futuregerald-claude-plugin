BINARY_NAME=skill-installer
VERSION=4.0.0

.PHONY: build clean test test-e2e install

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY_NAME) .

clean:
	rm -f $(BINARY_NAME)

test:
	go test ./...

test-e2e:
	bash scripts/run-e2e-docker.sh

install: build
	install -m 755 $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
