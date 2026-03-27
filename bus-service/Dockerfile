FROM golang:1.25.4-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download 

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/data ./data

EXPOSE 8083

CMD [ "./main" ]