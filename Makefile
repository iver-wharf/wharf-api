.PHONY: install tidy deps test \
	docker docker-run serve swag-force swag \
	lint lint-md lint-go \
	lint-fix lint-md-fix

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
	go install github.com/swaggo/swag/cmd/swag@v1.7.1
	go mod download
	npm install

test: swag
	go test -v ./...

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

swag:
ifeq ("$(wildcard docs/docs.go)","")
	swag init --parseDependency --parseDepth 1
else
ifeq ("$(filter $(MAKECMDGOALS),swag-force)","")
	@echo "-- Skipping 'swag init' because docs/docs.go exists."
	@echo "-- Run 'make' with additional target 'swag-force' to always run it."
endif
endif
	@# This comment silences warning "make: Nothing to be done for 'swag'."

lint: lint-md lint-go
lint-fix: lint-md-fix

lint-md:
	npx remark . .github

lint-md-fix:
	npx remark . .github -o

lint-go:
	revive -formatter stylish -config revive.toml ./...
