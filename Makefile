BINARY_NAME=skill-installer
VERSION=4.0.0

PYTEST_DIRS=\
	internal/catalog/shared/skills/editor-pass/scripts \
	internal/catalog/shared/skills/future-skill-creator/scripts \
	internal/catalog/shared/skills/grilling-page/scripts \
	internal/catalog/shared/skills/subagent-driven-development/scripts \
	internal/catalog/shared/skills/team-pulse/scripts \
	internal/catalog/shared/skills/visualize-plan/scripts \
	internal/catalog/antigravity/hooks

.PHONY: build clean test test-python test-e2e install

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY_NAME) .

clean:
	rm -f $(BINARY_NAME)

test:
	go test ./...
	PYTHONDONTWRITEBYTECODE=1 pytest -p no:cacheprovider -q $(PYTEST_DIRS)

test-python:
	PYTHONDONTWRITEBYTECODE=1 pytest -p no:cacheprovider -q $(PYTEST_DIRS)

test-e2e:
	bash scripts/run-e2e-docker.sh

install: build
	install -m 755 $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
