# syntax=docker/dockerfile:1

# Build stage. Frontend is built in CI and copied into webdist/ before this
# runs, so the go:embed directive in the serve command finds real files.
FROM golang:1.23.4-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /devotion ./cmd/devotion

# Runtime stage. Distroless-style minimal image; the binary serves TLS itself
# and embeds the frontend, so no web server or extra process is needed.
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 devotion
COPY --from=build /devotion /devotion
USER devotion
ENTRYPOINT ["/devotion"]
CMD ["serve"]
