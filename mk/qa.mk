.PHONY: qa
qa:
	@go fmt github.com/yunier-rojas/sombra-cli/internal/...
	@go fmt github.com/yunier-rojas/sombra-cli/cmd/...
	golangci-lint run
	@$(MAKE) imports
	@$(MAKE) lint-usecases
	@$(MAKE) lint-third-party
	@if [ -f sgconfig.yml ]; then $(MAKE) lint-sg; else echo "lint-sg: no sgconfig.yml, skipping"; fi
	@if [ -f mk/adr.mk ]; then $(MAKE) lint-adr; else echo "lint-adr: no mk/adr.mk, skipping"; fi
	@if [ -n "$(EXTRA_QA)" ]; then $(MAKE) $(EXTRA_QA); fi

.PHONY: imports
imports:
	@bash qa/check-imports.sh

.PHONY: lint-sg
lint-sg:
	ast-grep scan

.PHONY: lint-usecases
lint-usecases:
	go run ./qa/cmd/check-usecases

.PHONY: lint-third-party
lint-third-party:
	go run ./qa/cmd/check-third-party
