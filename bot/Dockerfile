FROM golang:1.26-alpine as build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./bot-service ./bot

FROM alpine:3.22 

WORKDIR /app

RUN adduser -D -u 11111 appuser

COPY --from=build --chown=appuser:appuser /app/bot-service .

USER appuser

EXPOSE 8080

CMD [ "./bot-service" ]