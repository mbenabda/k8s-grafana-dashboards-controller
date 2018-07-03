FROM golang:latest as builder 

RUN mkdir -p "$GOPATH/src/mbenabda.com/k8s-grafana-dashboards-controller" /dist
ADD . $GOPATH/src/mbenabda.com/k8s-grafana-dashboards-controller
WORKDIR $GOPATH/src/mbenabda.com/k8s-grafana-dashboards-controller

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh \
    && dep ensure -v \
    && CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s -w" -installsuffix nocgo -o /dist/binary . 


FROM scratch
COPY --from=builder /dist/binary /controller
ENTRYPOINT ["/controller"]