# ADR hygiene checks that tools like ast-grep cannot express.
#
#   make lint-adr
#
# Enforces the decisions in adr/process/2026-10-01-document-decisions-as-adrs.md:
# file names follow yyyy-mm-dd-short-title.md and the body stays under
# 30 lines and 500 words.

ADR_DIR := adr
ADR_RULE := $(ADR_DIR)/process/2026-10-01-document-decisions-as-adrs.md
ADR_MAX_LINES := 30
ADR_MAX_WORDS := 500

.PHONY: lint-adr
lint-adr:
	@if [ ! -d $(ADR_DIR) ]; then echo "adr: no $(ADR_DIR) directory, skipping"; exit 0; fi; \
	fail=0; \
	for f in $$(find $(ADR_DIR) -type f -name '*.md' | sort); do \
		parent=$$(basename $$(dirname "$$f")); \
		case "$$parent" in \
			product|process|technical) ;; \
			*) echo "adr: [$(ADR_RULE)] $$f must be in adr/product, adr/process or adr/technical"; fail=1; continue ;; \
		esac; \
		name=$$(basename "$$f"); \
		if ! printf '%s' "$$name" | grep -Eq '^[0-9]{4}-[0-9]{2}-[0-9]{2}-.+\.md$$'; then \
			echo "adr: [$(ADR_RULE)] bad file name: $$f (expected yyyy-mm-dd-short-title.md)"; \
			fail=1; \
		fi; \
		lines=$$(wc -l < "$$f" | tr -d ' '); \
		if [ "$$lines" -ge $(ADR_MAX_LINES) ]; then \
			echo "adr: [$(ADR_RULE)] $$f must be under $(ADR_MAX_LINES) lines (has $$lines)"; \
			fail=1; \
		fi; \
		words=$$(wc -w < "$$f" | tr -d ' '); \
		if [ "$$words" -ge $(ADR_MAX_WORDS) ]; then \
			echo "adr: [$(ADR_RULE)] $$f must be under $(ADR_MAX_WORDS) words (has $$words)"; \
			fail=1; \
		fi; \
	done; \
	if [ "$$fail" -ne 0 ]; then exit 1; fi
