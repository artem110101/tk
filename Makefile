BIN_DIR := $(HOME)/.local/bin

.PHONY: build install install-skill

build: bin/tk

bin/tk: $(wildcard *.go) go.mod go.sum build.sh
	./build.sh

install: build
	install -d "$(BIN_DIR)"
	install -m 0755 bin/tk "$(BIN_DIR)/tk"

install-skill: install
	@set -eC; \
	export PATH="$$PATH:$(BIN_DIR)"; \
	install_skill() { \
		skill="$$1/SKILL.md"; \
		if [ -e "$$skill" ] || [ -L "$$skill" ]; then \
			printf 'Keeping existing skill at %s\n' "$$skill"; \
			return; \
		fi; \
		install -d "$$1"; \
		sed "s|/absolute/path/to/tk|$(BIN_DIR)/tk|g" skills/tk/SKILL.md > "$$skill"; \
		printf 'Installed skill at %s\n' "$$skill"; \
	}; \
	if command -v claude >/dev/null 2>&1; then \
		install_skill "$(HOME)/.claude/skills/tk"; \
	fi; \
	if command -v codex >/dev/null 2>&1 || command -v pi >/dev/null 2>&1; then \
		install_skill "$(HOME)/.agents/skills/tk"; \
	fi
