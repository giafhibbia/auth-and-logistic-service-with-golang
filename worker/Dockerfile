FROM golang:1.22-alpine
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o main .
CMD ["./main"]
