FROM golang:1.20-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /consumidor-de-email

FROM alpine

WORKDIR /

COPY --from=build /consumidor-de-email /consumidor-de-email

ENTRYPOINT ["/consumidor-de-email"]
