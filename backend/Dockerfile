FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev git

# Copy all source code first
COPY . .

# Generate go.sum and download dependencies
ENV GONOSUMCHECK=*
ENV GOFLAGS=-mod=mod
RUN go mod tidy && go mod download

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/server

FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations

EXPOSE 3001

CMD ["./server"]
