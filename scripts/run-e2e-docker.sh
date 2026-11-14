#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

IMAGE_TAG="${SKILL_INSTALLER_E2E_IMAGE:-skill-installer-e2e:local}"

docker build -f e2e/Dockerfile -t "${IMAGE_TAG}" .
docker run --rm "${IMAGE_TAG}"

HOST_SANDBOX_HOME="$(mktemp -d)"
trap 'rm -rf "${HOST_SANDBOX_HOME}"' EXIT

HOST_GOROOT="$(go env GOROOT)"
HOST_GOPATH="$(go env GOPATH)"
HOST_GOCACHE="$(go env GOCACHE)"

PATH="${HOST_GOROOT}/bin:${PATH}" \
GOROOT="${HOST_GOROOT}" \
GOPATH="${HOST_GOPATH}" \
GOCACHE="${HOST_GOCACHE}" \
HOME="${HOST_SANDBOX_HOME}" \
go test -v -count=1 -tags=e2e ./e2e -run TestHostDesktopAppsCompatibilitySandbox
