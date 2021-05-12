# Build the manager binary
FROM golang:1.10.3 as builder

# Copy in the go src
WORKDIR /go/src/skiff-webconsole
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o webconsole main.go

# Copy the ripple into a thin image
FROM debian:stretch-slim
WORKDIR /
COPY --from=builder /go/src/skiff-webconsole/webconsole .
ENTRYPOINT ["/webconsole"]