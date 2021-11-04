ARG GOLANG_VERSION=1.17
ARG GOLANG_OPTIONS="CGO_ENABLED=0 GOOS=linux GOARCH=amd64"

FROM docker.io/golang:${GOLANG_VERSION} as build

ARG VERSION
ARG COMMIT

WORKDIR /project

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY cmd cmd
COPY cloudrun cloudrun
COPY consul consul
COPY generic generic

RUN env ${GOLANG_OPTIONS} \
    go build \
    -ldflags "-X 'main.BuildTime=${BUILD_TIME}' -X 'main.GitCommit=${COMMIT}' -X 'main.OSVersion=${VERSION}'" \
    -a -installsuffix cgo \
    -o /go/bin/discoverer \
    ./cmd

FROM gcr.io/distroless/base-debian10

LABEL org.opencontainers.image.source https://github.com/DazWilkin/consul-sd-cloudrun

COPY --from=build /go/bin/discoverer /

EXPOSE 9402

ENTRYPOINT ["/discoverer"]