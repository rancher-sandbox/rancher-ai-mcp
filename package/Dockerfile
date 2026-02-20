FROM registry.suse.com/bci/golang:1.24 AS builder
ARG VERSION=dev
ARG COMMIT
ARG LINKFLAGS="-extldflags -static -s"
ARG LDFLAGS="-X github.com/rancher/rancher-ai-mcp/pkg/version.Version=${VERSION} -X github.com/rancher/rancher-ai-mcp/pkg/version.GitCommit=${COMMIT} ${LINKFLAGS}"

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/
COPY main.go ./

RUN CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o mcp

FROM registry.suse.com/bci/bci-micro:15.7
ARG VERSION=dev

COPY --from=builder /src/mcp /mcp

LABEL "org.opencontainers.image.title"="Rancher MCP Server." \
    "org.opencontainers.image.description"="Rancher MCP Server allows the Rancher AI agent to securely retrieve or update Kubernetes and Rancher resources across local and downstream clusters." \
    "org.opencontainers.image.source"="https://github.com/rancher/rancher-ai-mcp" \
    "org.opencontainers.image.version"=${VERSION}

CMD ["/mcp"]
