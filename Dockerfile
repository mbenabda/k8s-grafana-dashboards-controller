FROM golang:latest as builder 

RUN mkdir -p "$GOPATH/src/mbenabda.com/k8s-grafana-dashboards-controller" /dist
ADD . $GOPATH/src/mbenabda.com/k8s-grafana-dashboards-controller
WORKDIR $GOPATH/src/mbenabda.com/k8s-grafana-dashboards-controller

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh \
    && dep ensure -v \
    && go test -v ./pkg/** \
    && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags "-s -w" -o /dist/binary ./cmd



FROM scratch
COPY --from=builder /dist/binary /controller
ENTRYPOINT ["/controller"]