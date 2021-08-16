FROM golang:1.16.5 AS build
WORKDIR /src
RUN go get -u github.com/swaggo/swag/cmd/swag@v1.7.0
COPY go.mod go.sum /src/
RUN go mod download

COPY . .
ARG BUILD_VERSION="local docker"
ARG BUILD_GIT_COMMIT="HEAD"
ARG BUILD_REF="0"
RUN deploy/update-version.sh version.yaml \
		&& swag init --parseDependency --parseDepth 1 \
		&& go get -t -d \
		&& CGO_ENABLED=0 go build -o main \
		&& go test -v ./...

FROM alpine:3.14.0 AS final
RUN apk add --no-cache ca-certificates
RUN apk add tzdata
WORKDIR /app
COPY --from=build /src/main ./
ENTRYPOINT ["/app/main"]
