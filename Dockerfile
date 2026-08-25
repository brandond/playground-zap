# syntax=docker/dockerfile:1.26-labs
FROM golang:1.25-alpine3.23 AS build
COPY --parents ./pkg ./go.mod ./go.sum ./main.go /go/src/github.com/brandond/playground-zap/
WORKDIR /go/src/github.com/brandond/playground-zap/
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    go build -o bin/playground-zap ./main.go

FROM busybox:1.38.0-uclibc AS image
COPY --from=build /go/src/github.com/brandond/playground-zap/bin/playground-zap /bin/playground-zap
WORKDIR /tmp
USER nobody
CMD ["/bin/playground-zap"]
