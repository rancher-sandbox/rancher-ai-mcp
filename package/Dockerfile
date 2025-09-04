FROM registry.suse.com/bci/golang:1.24 AS builder
WORKDIR /app
COPY .. .
RUN go build -o mcp

FROM registry.suse.com/bci/bci-micro:15.7
COPY --from=builder /app/mcp /mcp
CMD ["/mcp"]