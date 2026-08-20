FROM golang:1.26.7-bookworm AS build
WORKDIR /src/services/api
COPY services/api/go.mod services/api/go.sum ./
RUN go mod download
COPY services/api ./
ARG TARGETOS=linux
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install --no-install-recommends -y ca-certificates ffmpeg \
    && apt-get clean
RUN useradd --system --uid 65532 --create-home studio
WORKDIR /app
COPY --from=build /out/worker /app/worker
USER studio:studio
ENTRYPOINT ["/app/worker"]
