FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server .

# Distroless: no shell, no package manager, runs as UID 65532.
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/server /server
# Explicit numeric UID (the image's "nonroot" user) so Kubernetes and
# scanners can verify the container does not run as root.
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/server"]
