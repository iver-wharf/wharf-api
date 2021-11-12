FROM golang:1.16 AS build
WORKDIR /src
RUN go get -u github.com/swaggo/swag/cmd/swag@v1.7.1
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG BUILD_VERSION="local docker"
ARG BUILD_GIT_COMMIT="HEAD"
ARG BUILD_REF="0"
ARG BUILD_DATE=""
RUN deploy/update-version.sh version.yaml \
		&& make swag \
		&& CGO_ENABLED=0 go build -o main \
		&& make test

FROM alpine:3.14 AS final
RUN apk add --no-cache ca-certificates tzdata \
    && apk add --no-cache --upgrade \
        # CVE-2021-42374 & CVE 2021-42375: https://github.com/alpinelinux/docker-alpine/issues/213
        busybox>=1.33.1-r5
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
