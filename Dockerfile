ARG GOLANG_VERSION=1.23.0

ARG TARGETOS
ARG TARGETARCH

FROM --platform=${TARGETARCH} docker.io/golang:${GOLANG_VERSION} AS build

WORKDIR /project

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY cmd cmd
COPY cloudrun cloudrun
COPY consul consul
COPY generic generic

ARG TARGETOS
ARG TARGETARCH

ARG VERSION
ARG COMMIT

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build \
    -ldflags "-X 'main.BuildTime=${BUILD_TIME}' -X 'main.GitCommit=${COMMIT}' -X 'main.OSVersion=${VERSION}'" \
    -a -installsuffix cgo \
    -o /go/bin/discoverer \
    ./cmd


FROM --platform=${TARGETARCH} gcr.io/distroless/static-debian11:nonroot

LABEL org.opencontainers.image.source="https://github.com/DazWilkin/consul-sd-cloudrun"

COPY --from=build /go/bin/discoverer /

EXPOSE 9402

ENTRYPOINT ["/discoverer"]
