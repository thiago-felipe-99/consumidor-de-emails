FROM golang:1.20-alpine AS cache-file

WORKDIR /app

COPY go.sum ./
COPY go.mod ./
RUN go mod download

COPY . ./

FROM cache-file AS build-consumidor-de-emails

RUN go build -o /consumidor-de-emails

FROM cache-file AS build-benchmarking

RUN cd benchmarking && go build -o /benchmarking

FROM alpine

WORKDIR /

COPY --from=build-benchmarking         /benchmarking         /benchmarking
COPY --from=build-consumidor-de-emails /consumidor-de-emails /consumidor-de-emails

ENTRYPOINT ["/consumidor-de-emails"]
