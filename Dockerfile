# Build the manager binary
FROM ubuntu:22.04
ARG TARGETOS
ARG TARGETARCH
RUN ["apt-get", "update"]
RUN ["apt-get", "install", "-y", "wget"]
RUN ["apt-get", "install", "-y", "curl"]
RUN ["apt-get", "install", "-y", "vim"]
RUN ["apt-get", "install", "-y", "iproute2"]
RUN ["apt-get", "install", "-y", "nftables"]


WORKDIR /workspace

RUN ["wget", "https://dl.google.com/go/go1.20.linux-amd64.tar.gz"]
RUN ["tar", "-C", "/usr/local", "-xzf", "go1.20.linux-amd64.tar.gz"]
ENV PATH=$PATH:/usr/local/go/bin

# Install kubectl
RUN ["curl", "-LO", "https://storage.googleapis.com/kubernetes-release/release/v1.27.3/bin/linux/amd64/kubectl"]
RUN ["chmod", "+x", "./kubectl"]
RUN ["mv", "./kubectl", "/usr/local/bin/kubectl"]

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

# Copy the go source
COPY cmd/main.go main.go
COPY api/ api/
COPY internal/controller/ internal/controller/
COPY pkg/ pkg/

# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager main.go

CMD ["./manager"]

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
# FROM gcr.io/distroless/static:nonroot
# WORKDIR /
# COPY --from=builder /usr/local/bin/kubectl /usr/local/bin/kubectl
# USER 65532:65532

# ENTRYPOINT ["/manager"]
