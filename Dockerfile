# Build the manager binary
FROM golang:1.12.3 as builder

# Copy in the go src
WORKDIR /go/src/github.com/draios/internal-sysdig-labs/image-scan-k8s-webhook
COPY . /go/src/github.com/draios/internal-sysdig-labs/image-scan-k8s-webhook
RUN make test

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager /go/src/github.com/draios/internal-sysdig-labs/image-scan-k8s-webhook/cmd/manager

# Copy the controller-manager into a thin image
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /go/src/github.com/draios/internal-sysdig-labs/image-scan-k8s-webhook/manager .
ENTRYPOINT ["./manager"]
