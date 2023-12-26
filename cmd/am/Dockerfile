FROM golang:1.21-alpine3.18 as build

ADD ./go.mod /work/go.mod
ADD ./go.sum /work/go.sum
WORKDIR /work
RUN go mod download
ADD ./ /work
WORKDIR /work/cmd/am
RUN go mod download
RUN GOOS=linux GOARCH=amd64 go build

FROM alpine:3.18

RUN apk add --no-cache ca-certificates

COPY --from=build /work/cmd/am/am /usr/local/bin/am

ENTRYPOINT ["/usr/local/bin/am"]