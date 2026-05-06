FROM golang:1.26.2-alpine3.23 as build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./api-service ./api

FROM alpine:3.22 

WORKDIR /app

RUN adduser -D -u 11111 appuser

COPY --from=build --chown=appuser:appuser /app/api-service .

USER appuser

EXPOSE 8080

CMD [ "./api-service" ]