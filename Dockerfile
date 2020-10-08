FROM golang:alpine AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build -o scan-exporter .

FROM debian:buster-slim

WORKDIR /dist

COPY --from=builder /build/scan-exporter .

COPY --from=builder /build/config.yaml .

EXPOSE 2112

CMD [ "/dist/scan-exporter" ]
