FROM golang:1.23-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a static-ish binary for Alpine
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/syncthing-kicker ./cmd/syncthing-kicker


FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 65532 nonroot

WORKDIR /app

COPY --from=build /out/syncthing-kicker /usr/local/bin/syncthing-kicker

USER nonroot

ENTRYPOINT ["/usr/local/bin/syncthing-kicker"]
