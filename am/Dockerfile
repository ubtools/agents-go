FROM golang:1.21 as build

ADD ./ /work
WORKDIR /work
RUN cd /work/am && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /work/am/am

FROM alpine:3.14

RUN apk add --no-cache ca-certificates

COPY --from=build /work/am/am /usr/local/bin/am

ENTRYPOINT ["/usr/local/bin/am"]