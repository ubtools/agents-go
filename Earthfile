VERSION 0.7

build:
  ARG EARTHLY_TARGET_TAG_DOCKER
  ARG tag=$EARTHLY_TARGET_TAG_DOCKER
  BUILD ./cmd/agent-eth+build --tag $tag
  BUILD ./cmd/am+build --tag $tag
  BUILD ./cmd/am-pkcs+build --tag $tag

integration-tests:
  FROM ubtr/golang-nodejs:1.21.1-20.10-alpine3.18
  COPY . /work
  WORKDIR /work/tests
  RUN npm install
  RUN npm test

test:
  BUILD ./tests+test
  