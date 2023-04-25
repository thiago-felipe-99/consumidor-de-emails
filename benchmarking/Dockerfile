FROM golang:1.20-alpine AS build

WORKDIR /app

COPY ./benchmarking/go.sum ./
COPY ./benchmarking/go.mod ./
COPY ./rabbit ../rabbit

RUN go mod download

COPY ./benchmarking ./

RUN go build -o /benchmarking

FROM alpine

WORKDIR /

COPY --from=build /benchmarking /benchmarking

ENTRYPOINT ["/benchmarking"]
