.PHONY: install tidy deps check \
	docker docker-run serve swag-force swag proto \
	lint lint-md lint-go lint-proto \
	lint-fix lint-fix-md lint-fix-go lint-fix-proto

commit = $(shell git rev-parse HEAD)
version = latest

ifeq ($(OS),Windows_NT)
wharf-api.exe: swag
else
wharf-api: swag
endif
	go build .

install:
	go install

tidy:
	go mod tidy

deps:
	go install github.com/mgechev/revive@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/swaggo/swag/cmd/swag@v1.8.0
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1
	go install github.com/alta/protopatch/cmd/protoc-gen-go-patch@v0.5.0
	go install github.com/yoheimuta/protolint/cmd/protolint@v0.37.1
	go mod download
	npm install

check: swag
	go test ./...

docker:
	docker build . \
		--pull \
		-t "quay.io/iver-wharf/wharf-api:latest" \
		-t "quay.io/iver-wharf/wharf-api:$(version)" \
		--build-arg BUILD_VERSION="$(version)" \
		--build-arg BUILD_GIT_COMMIT="$(commit)" \
		--build-arg BUILD_DATE="$(shell date --iso-8601=seconds)"
	@echo ""
	@echo "Push the image by running:"
	@echo "docker push quay.io/iver-wharf/wharf-api:latest"
ifneq "$(version)" "latest"
	@echo "docker push quay.io/iver-wharf/wharf-api:$(version)"
endif

docker-run:
	docker run --rm -it quay.io/iver-wharf/wharf-api:$(version)

serve: swag
	go run .

clean:
	@rm -vf docs/docs.go
	@rm -vf docs/swagger.json
	@rm -vf docs/swagger.yaml

swag-force:
	swag init --parseDependency --parseDepth 1

swag: docs/docs.go

docs/docs.go:
	swag init --parseDependency --parseDepth 1

proto:
	protoc -I . \
		-I `go list -m -f {{.Dir}} github.com/alta/protopatch` \
		-I `go list -m -f {{.Dir}} google.golang.org/protobuf` \
		--go-patch_out=plugin=go,paths=source_relative:. \
		--go-patch_out=plugin=go-grpc,paths=source_relative:. \
		./api/wharfapi/v5/builds.proto
# Generated files have some non-standard formatting, so let's format it.
	goimports -w ./api/wharfapi/v5/.

lint: lint-md lint-go lint-proto
lint-fix: lint-fix-md lint-fix-go lint-fix-proto

lint-md:
	npx remark . .github

lint-fix-md:
	npx remark . .github -o

lint-go:
	goimports -d $(shell git ls-files "*.go")
	revive -formatter stylish -config revive.toml ./...

lint-fix-go:
	goimports -d -w $(shell git ls-files "*.go")

lint-proto:
	protolint lint api/wharfapi

lint-fix-proto:
	protolint lint -fix api/wharfapi
