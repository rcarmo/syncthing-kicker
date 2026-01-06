FROM golang:1.23-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a static binary
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/syncthing-kicker ./cmd/syncthing-kicker


FROM gcr.io/distroless/static-debian12:nonroot

# Keep HTTPS verification working and support CRON_TZ/TZ timezones.
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo

COPY --from=build /out/syncthing-kicker /syncthing-kicker

ENTRYPOINT ["/syncthing-kicker"]
