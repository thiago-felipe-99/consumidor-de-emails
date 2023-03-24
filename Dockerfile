FROM golang:1.20-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . ./

RUN go build -o /consumidor-de-email
RUN cd benchmarking && go build -o /benchmarking

FROM alpine

WORKDIR /

COPY --from=build /consumidor-de-email /consumidor-de-email
COPY --from=build /benchmarking /benchmarking

ENTRYPOINT ["/consumidor-de-email"]
