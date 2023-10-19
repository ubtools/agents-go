FROM golang:1.21 as build

ADD ./ /work
WORKDIR /work
RUN cd /work/cmd/agent-eth && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build

FROM alpine:3.14

RUN apk add --no-cache ca-certificates

COPY --from=build /work/cmd/agent-eth/agent-eth /usr/local/bin/agent-eth

ENTRYPOINT ["/usr/local/bin/agent-eth"]