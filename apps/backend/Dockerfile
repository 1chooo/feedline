FROM golang:1.25-alpine

RUN apk add --no-cache ca-certificates git netcat-openbsd \
  && go install github.com/air-verse/air@v1.67.1

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]
