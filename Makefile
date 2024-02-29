VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT  := $(shell git log -1 --format='%H')
DIRTY := $(shell git status --porcelain | wc -l | xargs)

ldflags = -X github.com/strangelove-ventures/noble-cctp-relayer/cmd.Version=$(VERSION) \
				-X github.com/strangelove-ventures/noble-cctp-relayer/cmd.Commit=$(COMMIT) \
				-X github.com/strangelove-ventures/noble-cctp-relayer/cmd.Dirty=$(DIRTY)

ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

.PHONY: all format lint
all: format lint

###############################################################################
###                          Formatting & Linting                           ###
###############################################################################

gofumpt_cmd=mvdan.cc/gofumpt
golangci_lint_cmd=github.com/golangci/golangci-lint/cmd/golangci-lint

format:
	@echo "🤖 Running formatter..."
	@go run $(gofumpt_cmd) -l -w .
	@echo "✅ Completed formatting!"

lint:
	@echo "🤖 Running linter..."
	@go run $(golangci_lint_cmd) run --timeout=10m
	@echo "✅ Completed linting!"


###############################################################################
###                              Install                                    ###
###############################################################################

install: go.sum
	@echo "🤖 Building noble-cctp-relayer..."
	@go build -mod=readonly -ldflags '$(ldflags)' -o $(GOBIN)/noble-cctp-relayer main.go
