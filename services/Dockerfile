# One build stage for every service; pick one with --target <service>.
ARG IMAGE_REGISTRY=docker.io
ARG GO_TAG=1.24.9-alpine
FROM --platform=$BUILDPLATFORM ${IMAGE_REGISTRY}/library/golang:${GO_TAG} AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/...

FROM scratch AS inventory
COPY --from=build /out/inventory /inventory
COPY --from=build /out/labctl /labctl
EXPOSE 8080
ENTRYPOINT ["/inventory"]

FROM scratch AS orders
COPY --from=build /out/orders /orders
COPY --from=build /out/labctl /labctl
EXPOSE 8080
ENTRYPOINT ["/orders"]

FROM scratch AS loadgen
COPY --from=build /out/loadgen /loadgen
ENTRYPOINT ["/loadgen"]
