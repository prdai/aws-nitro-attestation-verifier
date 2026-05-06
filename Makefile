GO_MODULES := client ecs nitro-enclave

.PHONY: hooks-install go-fmt go-fmt-check go-lint go-check infra-init infra-fmt infra-fmt-check infra-validate infra-check infra-plan infra-deploy infra-destroy infra-state

hooks-install:
	git config core.hooksPath .githooks

go-fmt:
	@for module in $(GO_MODULES); do \
		echo "golangci-lint fmt $$module"; \
		(cd $$module && golangci-lint fmt --config ../.golangci.yml ./...); \
	done

go-fmt-check:
	@for module in $(GO_MODULES); do \
		echo "golangci-lint fmt --diff $$module"; \
		(cd $$module && golangci-lint fmt --config ../.golangci.yml --diff ./...); \
	done

go-lint:
	@for module in $(GO_MODULES); do \
		if find $$module -name '*.go' -type f | grep -q .; then \
			echo "golangci-lint run $$module"; \
			(cd $$module && golangci-lint run --config ../.golangci.yml ./...); \
		else \
			echo "golangci-lint run $$module skipped: no Go files"; \
		fi; \
	done

go-check: go-fmt-check go-lint

infra-init:
	terraform -chdir=infra init

infra-fmt:
	terraform -chdir=infra fmt

infra-fmt-check:
	terraform -chdir=infra fmt -check -recursive

infra-validate:
	terraform -chdir=infra validate

infra-check: infra-fmt-check infra-validate infra-state

infra-plan:
	terraform -chdir=infra plan

infra-deploy:
	terraform -chdir=infra apply

infra-destroy:
	terraform -chdir=infra destroy

infra-state:
	terraform -chdir=infra state list
