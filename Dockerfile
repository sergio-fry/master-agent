# Multi-stage build: Go binary + openssh-client only (no Cursor/MCP/backlog).
# Secrets (SSH keys, known_hosts) must be mounted at runtime — never COPY'd.

FROM golang:1.26-alpine AS build
WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
	-o /out/master-agent ./cmd/master-agent

FROM alpine:3.22
RUN apk add --no-cache ca-certificates openssh-client \
	&& mkdir -p /data /secrets \
	&& chmod 755 /data /secrets

COPY --from=build /out/master-agent /usr/local/bin/master-agent

VOLUME ["/data"]

ENTRYPOINT ["master-agent"]
CMD ["daemon"]
