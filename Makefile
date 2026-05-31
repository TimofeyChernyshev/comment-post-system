COVERAGE_FILE ?= coverage.out

## test: run all tests
.PHONY: test
test:
	@go test -coverpkg='github.com/TimofeyChernyshev/comment-post-system/...' --race -count=1 -coverprofile='$(COVERAGE_FILE)' ./...
	@go tool cover -func='$(COVERAGE_FILE)' | grep ^total | tr -s '\t'
