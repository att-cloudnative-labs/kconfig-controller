FROM docker.io/golang:1.11 as buildimage

COPY cmd $GOPATH/src/github.com/gbraxton/kconfig/cmd
COPY internal $GOPATH/src/github.com/gbraxton/kconfig/internal
COPY pkg $GOPATH/src/github.com/gbraxton/kconfig/pkg
COPY vendor $GOPATH/src/github.com/gbraxton/kconfig/vendor
WORKDIR $GOPATH/src/github.com/gbraxton/kconfig/

# minimal passwd file is created for user in scratch image
# Empty file is needed to create /tmp directory in scratch image
RUN go get -d -v ./... && \
    CGO_ENABLED=0 go build -o /go/bin/kconfig-controller github.com/gbraxton/kconfig/cmd && \
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
