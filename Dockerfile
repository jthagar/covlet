FROM golang:125 AS base

WORKDIR /app
COPY go.* ./

RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /covlet

FROM scratch

COPY --from=base /app/covlet ./covlet
EXPOSE 3000

CMD ["covlet"]

