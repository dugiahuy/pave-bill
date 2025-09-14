ifeq ($(GO),)
GO = $(shell command -v go1.23 2> /dev/null)
endif
ifeq ($(GO),)
GO = $(shell command -v go 2> /dev/null)
endif
ifeq ($(GO),)
$(error "Couldn't find go, make sure you have installed go.")
endif

MOCKGEN ?= $(shell command -v mockgen 2> /dev/null)

.PHONY: help install-tools generate-mocks test test-coverage clean

help:
	@echo "Available targets:"
	@echo "  install-tools    - Install development tools (mockgen, etc.)"
	@echo "  generate-mocks   - Generate all mock files"
	@echo "  test            - Run all tests"
	@echo "  test-coverage   - Run tests with coverage"
	@echo "  clean           - Clean generated files"

install-tools:
	@echo "Installing development tools..."
	go install go.uber.org/mock/mockgen@latest

generate-mocks:
	@echo "Generating mocks..."
	# Generate repository interface mocks
	mockgen -source=billing/repository/currencies/querier.go -destination=billing/mocks/repository/currency_repo/mock.go -package=currency_repo
	mockgen -source=billing/repository/bills/querier.go -destination=billing/mocks/repository/bill_repo/mock.go -package=bill_repo
	mockgen -source=billing/repository/lineitems/querier.go -destination=billing/mocks/repository/lineitem_repo/mock.go -package=lineitem_repo
	# Generate business interface mocks
	mockgen -source=billing/business/bill/business.go -destination=billing/mocks/business/bill_business/mock.go -package=bill_business
	mockgen -source=billing/business/currency/business.go -destination=billing/mocks/business/currency_business/mock.go -package=currency_business
	# Generate domain interface mocks
	mockgen -source=billing/domain/bill_state_machine/bill_state_machine.go -destination=billing/mocks/domain/state_machine/mock.go -package=state_machine
	@echo "Mocks generated successfully!"

# Run all tests
test:
	go test ./... -v

# Run tests with coverage
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean generated files
clean:
	rm -rf billing/mocks
	rm -f coverage.out coverage.html

# Install tools and generate mocks in one command
setup: install-tools generate-mocks
	@echo "Setup complete!"
