# API server only (TUI runs on the host: go run ./frontend/cmd/covlet-tui).
FROM golang:1.23-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/covlet ./backend/cmd/covlet

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/covlet /covlet
EXPOSE 8080
USER nonroot:nonroot
ENV COVLET_LISTEN=:8080
CMD ["/covlet"]
