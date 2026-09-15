.PHONY: qa
qa:
	@go fmt github.com/yunier-rojas/sombra-cli/internal/...
	@go fmt github.com/yunier-rojas/sombra-cli/cmd/...
	golangci-lint run


.PHONY: imports
imports:
	go run github.com/quantumcycle/go-import-checks@latest --config qa/.import.yaml
