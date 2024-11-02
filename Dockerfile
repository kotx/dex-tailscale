FROM golang:1.23 AS build

WORKDIR /go/src/app
COPY go.mod ./cmd .

RUN CGO_ENABLED=0 go build -o /go/bin/proxy ./cmd/proxy

FROM gcr.io/distroless/static-debian12

COPY --from=build /go/bin/proxy /
CMD ["/proxy"]
