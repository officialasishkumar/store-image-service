FROM golang:1.20-alpine AS builder

ENV GO111MODULE=off

WORKDIR /app

COPY . .

RUN go build -o server .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/server .

COPY --from=builder /app/StoreMasterAssignment.csv .

EXPOSE 8080

CMD ["./server"]
