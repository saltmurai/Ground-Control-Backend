FROM golang:1.20.4-alpine3.16

RUN mkdir /app

RUN go install github.com/cosmtrek/air@latest



WORKDIR /app

COPY . /app

RUN go mod tidy
