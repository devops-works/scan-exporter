# Get last release from GitHub
FROM alpine:latest AS fetcher

RUN apk add curl

WORKDIR /build

RUN curl -s https://api.github.com/repos/devops-works/scan-exporter/releases/latest \
    | grep "browser_download_url.*linux_amd64" \
    | cut -d : -f 2,3 \
    | tr -d \" > release

RUN wget $(cat release) && \
    chmod +x scan-exporter_linux_amd64

RUN curl -O https://raw.githubusercontent.com/devops-works/scan-exporter/master/config-sample.yaml

# Run the app
FROM gcr.io/distroless/base-debian10

WORKDIR /app

COPY --from=fetcher /build/scan-exporter_linux_amd64 scan-exporter

COPY --from=fetcher /build/config-sample.yaml config.yaml

EXPOSE 2112

ENTRYPOINT [ "/app/scan-exporter" ]
