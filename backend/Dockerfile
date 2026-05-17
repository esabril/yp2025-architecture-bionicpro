FROM golang:1.22-alpine AS build

WORKDIR /src

COPY go.mod ./
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /reports-api ./cmd/reports-api

FROM alpine:3.20

WORKDIR /app

COPY --from=build /reports-api /app/reports-api

EXPOSE 8000

CMD ["/app/reports-api"]
