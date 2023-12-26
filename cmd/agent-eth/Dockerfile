FROM golang:1.21-alpine3.18 as build

ADD ./go.mod /work/go.mod
ADD ./go.sum /work/go.sum
WORKDIR /work
RUN go mod download
ADD ./ /work
WORKDIR /work/cmd/agent-eth

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build

FROM alpine:3.18

RUN apk add --no-cache ca-certificates

COPY --from=build /work/cmd/agent-eth/agent-eth /usr/local/bin/agent-eth
COPY --from=build /work/cmd/agent-eth/config.default.yaml /etc/agent-eth/config.yaml

ENTRYPOINT ["/usr/local/bin/agent-eth"]
CMD ["-c", "/etc/agent-eth/config.yaml"]