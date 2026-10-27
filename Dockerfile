# Match go/go.mod's toolchain; digests pin the verified multi-platform indexes.
FROM --platform=$BUILDPLATFORM golang:1.24.5-bookworm@sha256:ef8c5c733079ac219c77edab604c425d748c740d8699530ea6aced9de79aea40 AS build
WORKDIR /src
ENV GOTOOLCHAIN=local
COPY go/go.mod go/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY go/ ./
ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/k8sautodocs .

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
WORKDIR /app
COPY --from=build /out/k8sautodocs ./k8sautodocs
COPY go/static/ ./static/
LABEL org.opencontainers.image.source="https://github.com/tintii/goK8sAutoDocs" \
      org.opencontainers.image.description="Kubernetes versions and endpoint documentation dashboards"
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/app/k8sautodocs"]
