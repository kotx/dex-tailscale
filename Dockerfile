FROM --platform=${BUILDPLATFORM} \
    golang:1.23 AS build

WORKDIR /go/src/app

COPY go.mod go.sum .
RUN go mod download && go mod verify

COPY ./cmd/proxy ./cmd/proxy
RUN go vet -v ./cmd/proxy

RUN CGO_ENABLED=0 go build -o /go/bin/proxy ./cmd/proxy

FROM gcr.io/distroless/static-debian12

COPY --from=build /go/bin/proxy /
CMD ["/proxy"]
