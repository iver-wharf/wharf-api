ARG REG=docker.io
FROM ${REG}/library/golang:1.17 AS build
WORKDIR /src
RUN go install github.com/swaggo/swag/cmd/swag@v1.7.1
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG BUILD_VERSION="local docker"
ARG BUILD_GIT_COMMIT="HEAD"
ARG BUILD_REF="0"
ARG BUILD_DATE=""
RUN chmod +x deploy/update-version.sh  \
    && deploy/update-version.sh version.yaml \
    && make swag check \
    && CGO_ENABLED=0 go build -o main

ARG REG=docker.io
FROM ${REG}/library/alpine:3.14 AS final
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /src/main ./
ENTRYPOINT ["/app/main"]

ARG BUILD_VERSION
ARG BUILD_GIT_COMMIT
ARG BUILD_REF
ARG BUILD_DATE
# The added labels are based on this: https://github.com/projectatomic/ContainerApplicationGenericLabels
LABEL name="iver-wharf/wharf-api" \
    url="https://github.com/iver-wharf/wharf-api" \
    release=${BUILD_REF} \
    build-date=${BUILD_DATE} \
    vendor="Iver" \
    version=${BUILD_VERSION} \
    vcs-type="git" \
    vcs-url="https://github.com/iver-wharf/wharf-api" \
    vcs-ref=${BUILD_GIT_COMMIT} \
    changelog-url="https://github.com/iver-wharf/wharf-api/blob/${BUILD_VERSION}/CHANGELOG.md" \
    authoritative-source-url="quay.io"
