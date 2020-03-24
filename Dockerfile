# Build the manager binary
FROM golang:1.14.1 as builder
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
# Copy in the go src
WORKDIR /go/src/github.com/draios/internal-sysdig-labs/image-scan-k8s-webhook
COPY . /go/src/github.com/draios/internal-sysdig-labs/image-scan-k8s-webhook
RUN make deps test

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager /go/src/github.com/draios/internal-sysdig-labs/image-scan-k8s-webhook/cmd/manager

# Copy the controller-manager into a thin image
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /go/src/github.com/draios/internal-sysdig-labs/image-scan-k8s-webhook/manager .
ENTRYPOINT ["./manager"]
