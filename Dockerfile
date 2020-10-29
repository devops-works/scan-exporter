FROM devopsworks/golang-upx:1.15 AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build -o scan-exporter . && \
    strip scan-exporter && \
    /usr/local/bin/upx -9 scan-exporter

RUN setcap cap_net_raw+ep scan-exporter

FROM gcr.io/distroless/base-debian10

WORKDIR /app

COPY --from=builder /build/scan-exporter .

COPY --from=builder /build/config-sample.yaml config.yaml

EXPOSE 2112

ENTRYPOINT [ "/app/scan-exporter" ]
