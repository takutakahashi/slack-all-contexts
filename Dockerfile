# Build stage
FROM golang:1.19-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN GOOS=linux go build -o slack-all-contexts .

# Final stage
FROM gcr.io/distroless/base-debian11

COPY --from=builder /app/slack-all-contexts /slack-all-contexts

EXPOSE 8080

ENTRYPOINT ["/slack-all-contexts"]
