.PHONY: infra-init infra-fmt infra-validate infra-plan infra-deploy infra-destroy infra-state

infra-init:
	terraform -chdir=infra init

infra-fmt:
	terraform -chdir=infra fmt

infra-validate:
	terraform -chdir=infra validate

infra-plan:
	terraform -chdir=infra plan

infra-deploy:
	terraform -chdir=infra apply

infra-destroy:
	terraform -chdir=infra destroy

infra-state:
	terraform -chdir=infra state list
