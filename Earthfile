VERSION 0.7

build:
  ARG EARTHLY_TARGET_TAG_DOCKER
  ARG tag=$EARTHLY_TARGET_TAG_DOCKER
  BUILD ./cmd/agent-eth+build --tag $tag
  BUILD ./cmd/am+build --tag $tag
  BUILD ./cmd/am-pkcs+build --tag $tag

it:
  FROM ubtr/golang-nodejs:1.21.1-20.10-alpine3.18
  COPY . /work
  WORKDIR /work/it
  RUN npm install
  RUN npm test

test:
  LOCALLY
  RUN go test -v ./...

ci:
  FROM ubtr/golang-nodejs:1.21.1-20.10-alpine3.18
  COPY . /work
  WORKDIR /work
  RUN CGO_ENABLED=1 go test -v ./...
  WORKDIR /work/it
  RUN npm install
  RUN npm test

release:
  ARG EARTHLY_TARGET_TAG_DOCKER
  ARG repo=ubtr
  FROM alpine:3.18
  ARG tag=$(echo -n ${EARTHLY_TARGET_TAG_DOCKER} | sed 's/v\(.*\)/\1/')
  BUILD +build --tag $tag --repo $repo
  