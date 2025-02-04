lint:
	golangci-lint run

test-unit:
	go test ./internal/... -race -coverprofile=coverage.out -covermode=atomic -v

test-integration:
	go test -tags integration ./internal/handler/... -race -coverprofile=coverage_integration.out -coverpkg=./internal/handler/... -covermode=atomic -v

doc:
	swag init --parseDependency -g internal/route/route.go -o docs
