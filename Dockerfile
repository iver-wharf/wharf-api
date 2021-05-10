FROM golang:1.16.3 AS build
WORKDIR /src
RUN go get -u github.com/swaggo/swag/cmd/swag@v1.7.0
COPY go.mod go.sum /src/
RUN go mod download
COPY . /src
RUN swag init && go get -t -d && CGO_ENABLED=0 go build -o main && go test -v ./...

FROM alpine:3.13.4 AS final
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /src/main ./
ENTRYPOINT ["/app/main"]
