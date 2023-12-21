FROM golang:1.21 as build

ADD ./ /work
WORKDIR /work
RUN cd /work/am-pkcs && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /work/am-pkcs/am-pkcs

FROM alpine:3.14

RUN apk add --no-cache ca-certificates

COPY --from=build /work/am-pkcs/am-pkcs /usr/local/bin/am-pkcs

ENTRYPOINT ["/usr/local/bin/am-pkcs"]