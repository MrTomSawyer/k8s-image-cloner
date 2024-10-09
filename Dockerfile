FROM golang:1.22.5 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -o cloner ./cmd/cloner/main.go

FROM gcr.io/distroless/base-debian10
WORKDIR /
COPY --from=builder /app/cloner .
USER nonroot:nonroot
ENTRYPOINT ["/cloner"]
