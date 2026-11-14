FROM golang:1.27-bookworm AS go-toolchain

FROM node:22-bookworm

COPY --from=go-toolchain /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"

RUN apt-get update && apt-get install -y --no-install-recommends \
    bash \
    ca-certificates \
    git \
    python3 \
    python3-pytest \
    python3-yaml \
    && rm -rf /var/lib/apt/lists/*

RUN npm install -g @anthropic-ai/claude-code opencode-ai

RUN printf '#!/usr/bin/env bash\necho "Zed 0.175.0 (e2e-container)"\n' > /usr/local/bin/zed && \
    chmod 0755 /usr/local/bin/zed && \
    printf '#!/usr/bin/env bash\necho "Google Antigravity 1.0.0 (e2e-container)"\n' > /usr/local/bin/antigravity && \
    chmod 0755 /usr/local/bin/antigravity

RUN useradd -m -s /bin/bash e2euser

WORKDIR /workspace
COPY go.mod go.sum /workspace/
RUN go mod download

COPY . /workspace

RUN go build -ldflags "-X main.version=4.0.0" -o /usr/local/bin/skill-installer . && \
    chown -R e2euser:e2euser /workspace

USER e2euser
ENV SKILL_INSTALLER_E2E_CONTAINER=1
ENV SKILL_INSTALLER_BIN=/usr/local/bin/skill-installer

CMD ["go", "test", "-v", "-count=1", "-tags=e2e", "./e2e"]
