# Stage 1: build the static binary
FROM golang:1.26.2-trixie AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -X main.version=${VERSION}" -o /bin/mikrom .

# Stage 2: minimal runtime — distroless/static is ideal for a CGO-free CLI
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /bin/mikrom /usr/local/bin/mikrom

ENTRYPOINT ["/usr/local/bin/mikrom"]
