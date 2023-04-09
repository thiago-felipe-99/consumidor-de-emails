FROM golang:1.20-alpine AS build

WORKDIR /app

COPY go.sum ./
COPY go.mod ./
RUN go mod download

COPY ./benchmarking/ ./

RUN go build -o /benchmarking

FROM alpine

WORKDIR /

COPY --from=build /benchmarking /benchmarking

ENTRYPOINT ["/benchmarking"]
