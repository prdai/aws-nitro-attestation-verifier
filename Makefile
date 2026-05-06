.PHONY: hooks-install infra-init infra-fmt infra-fmt-check infra-validate infra-check infra-plan infra-deploy infra-destroy infra-state

hooks-install:
	git config core.hooksPath .githooks

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
