FROM docker.io/golang:1.12.4 as buildimage

COPY cmd $GOPATH/src/github.com/att-cloudnative-labs/kconfig-controller/cmd
COPY internal $GOPATH/src/github.com/att-cloudnative-labs/kconfig-controller/internal
COPY pkg $GOPATH/src/github.com/att-cloudnative-labs/kconfig-controller/pkg
COPY go.mod $GOPATH/src/github.com/att-cloudnative-labs/kconfig-controller/go.mod
COPY go.sum $GOPATH/src/github.com/att-cloudnative-labs/kconfig-controller/go.sum
WORKDIR $GOPATH/src/github.com/att-cloudnative-labs/kconfig-controller

# minimal passwd file is created for user in scratch image
# Empty file is needed to create /tmp directory in scratch image
RUN GO111MODULE=on CGO_ENABLED=0 go build -o /go/bin/kconfig-controller ./cmd && \
    echo "kconfig:x:1001:1001::/:" > /tmp/passwd.minimal && \
    touch /tmp/.empty

FROM scratch
# This new passwd file contains kconfig user
COPY --from=buildimage /tmp/passwd.minimal /etc/passwd
USER kconfig
# Create /tmp directory for log using empty file
COPY --from=buildimage /tmp/.empty /tmp/.empty
COPY --from=buildimage /go/bin/kconfig-controller /usr/bin/kconfig-controller
ENTRYPOINT ["/usr/bin/kconfig-controller"]
