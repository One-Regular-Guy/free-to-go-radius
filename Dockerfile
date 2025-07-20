FROM golang:1.24.5-bookworm

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o main .

CMD ["./main"]

EXPOSE 1812