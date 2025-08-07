FROM devopsworks/golang-upx:1.24.6 AS builder

RUN apt-get update && apt-get install -y libcap2-bin

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

ARG VERSION="n/a"
ARG BUILD_DATE="n/a"

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

# RUN go build -o scan-exporter . && \
#     strip scan-exporter && \
#     /usr/local/bin/upx -9 scan-exporter

RUN go build \
    -ldflags "-X main.Version=${version} -X main.BuildDate=${builddate}" \
    -o scan-exporter . && \
    strip scan-exporter && \
    /usr/local/bin/upx -9 scan-exporter

RUN setcap cap_net_raw+ep scan-exporter

FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY --from=builder /build/scan-exporter .

COPY --from=builder /build/config-sample.yaml config.yaml

EXPOSE 2112

ENTRYPOINT [ "/app/scan-exporter" ]
