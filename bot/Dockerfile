FROM golang:1.26.2-alpine3.23 as build

WORKDIR /app

RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -v -o ./bot-service ./bot

FROM alpine:3.22 

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -u 11111 appuser && mkdir /data && chown appuser:appuser /data

COPY --from=build --chown=appuser:appuser /app/bot-service .

USER appuser

EXPOSE 8080

CMD [ "./bot-service" ]
