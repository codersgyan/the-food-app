# Build the three programs in a Go image, then copy only the programs into a
# small image with no shell and no package manager, so there's very little in
# it an attacker could use.

FROM golang:1.27.3 AS build
WORKDIR /src

# Download the dependencies first, so Docker can reuse this layer until
# go.mod or go.sum changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=0 makes programs that need nothing from the operating system,
# so they run in the empty image below.
RUN CGO_ENABLED=0 go build -o /out/food-api ./cmd/api && \
    CGO_ENABLED=0 go build -o /out/migrate ./cmd/migrate && \
    CGO_ENABLED=0 go build -o /out/seed ./cmd/seed

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/ /usr/local/bin/
# Run as a user with no rights, not as root.
USER nonroot
EXPOSE 8080
# The image starts the API server. To run another program in it:
#     docker compose -f compose.prod.yaml run --rm api migrate up
CMD ["food-api"]
