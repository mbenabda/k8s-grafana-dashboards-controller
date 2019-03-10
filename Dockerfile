FROM golang:1.10.8-stretch as builder 

ENV DEP_RELEASE_TAG=v0.5.0 \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

RUN mkdir -p "$GOPATH/src/mbenabda.com/k8s-grafana-dashboards-controller" /dist
ADD . $GOPATH/src/mbenabda.com/k8s-grafana-dashboards-controller
WORKDIR $GOPATH/src/mbenabda.com/k8s-grafana-dashboards-controller

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh \
    && dep ensure -v \
    && go test -v ./pkg/** \
    && go build -a -ldflags "-s -w" -o /dist/binary ./cmd



FROM scratch
COPY --from=builder /dist/binary /controller
ENTRYPOINT ["/controller"]