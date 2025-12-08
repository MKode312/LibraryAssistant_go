FROM golang:1.25.3-alpine AS builder

WORKDIR /app

RUN apk --no-cache add bash gcc musl-dev

COPY ./go.mod ./go.sum ./

RUN go mod download

COPY ./ ./

RUN CGO_ENABLED=1 go build -o /app/api ./cmd/LibAssistant-api/main.go

FROM alpine:3.22.2

WORKDIR /app

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /app/api .

RUN chown -R appuser:appgroup /app

USER appuser 

EXPOSE 8082

CMD ["./api"]