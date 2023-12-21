FROM golang:1.21-alpine3.18 as build

RUN apk add --no-cache gcc musl-dev
ADD ./ /work
WORKDIR /work/cmd/am-pkcs
RUN go mod download
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build

FROM alpine:3.18

RUN apk add --no-cache ca-certificates

COPY --from=build /work/cmd/am-pkcs/am-pkcs /usr/local/bin/am-pkcs

ENTRYPOINT ["/usr/local/bin/am-pkcs"]