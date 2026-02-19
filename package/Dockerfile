FROM registry.suse.com/bci/golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 go build -o mcp

FROM registry.suse.com/bci/bci-micro:15.7
COPY --from=builder /app/mcp /mcp
CMD ["/mcp"]
