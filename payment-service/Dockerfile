FROM golang:1.25.4-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download 

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/main .

EXPOSE 8085
EXPOSE 50051

CMD [ "./main" ]
