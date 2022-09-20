FROM golang:alpine3.16 AS builder
WORKDIR /build
ADD go.mod .
COPY . .
RUN GOPROXY=off go build -o app -mod=vendor main.go

FROM alpine
WORKDIR /app 
COPY --from=builder /build/app /app/app
CMD ["./app"]