.PHONY: ci tidy-check generate generate-check fmt fmt-check lint vuln test coverage

ci: tidy-check generate-check fmt-check lint vuln test coverage

tidy-check:
	go mod tidy
	git diff --exit-code -- go.mod go.sum

generate:
	go generate ./tests/apps/basic
	go generate ./tests/apps/basic/openapiclient
	go generate ./tests/apps/middleware

generate-check: generate
	git diff --exit-code -- tests/apps

fmt:
	go tool goimports -w main.go internal tests

fmt-check: fmt
	git diff --exit-code -- main.go internal tests

lint:
	go tool golangci-lint run ./...

vuln:
	go tool govulncheck ./...

test:
	go test ./...

coverage:
	@packages="$$(go list ./... | grep -v '/tests/apps/' | tr '\n' ' ')"; \
	go test $$packages -coverprofile=coverage.out -covermode=atomic; \
	go tool cover -func=coverage.out; \
	total="$$(go tool cover -func=coverage.out | awk '/^total:/ { sub(/%/, "", $$3); print $$3 }')"; \
	awk -v total="$$total" 'BEGIN { if (total < 95) exit 1 }'
