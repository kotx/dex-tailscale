FROM golang:1.23

WORKDIR /usr/src/app

COPY go.mod ./
# # pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
# COPY go.mod go.sum ./
# RUN go mod download && go mod verify

COPY ./cmd .
RUN go build -v -o /usr/local/bin/proxy ./cmd/proxy

CMD ["proxy"]
