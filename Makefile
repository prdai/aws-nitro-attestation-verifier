GO_MODULES := client ec2 nitro-enclave
AWS_NITRO_ROOT_CERT_DIR := .cache/aws-nitro-root
AWS_NITRO_ROOT_CERT := $(AWS_NITRO_ROOT_CERT_DIR)/AWS_NitroEnclaves_Root-G1.pem
AWS_NITRO_ROOT_CERT_ZIP_ENTRY ?= root.pem
ENCLAVE_IMAGE ?= nitro-attestation-enclave:latest
ENCLAVE_EIF ?= nitro-attestation-enclave.eif
ENCLAVE_NAME ?= nitro-attestation-enclave
ENCLAVE_CID ?= 16
ENCLAVE_PORT ?= 5000
ENCLAVE_MEMORY_MIB ?= 512
ENCLAVE_CPU_COUNT ?= 1
NITRO_CLI_BLOBS ?= /usr/share/nitro_enclaves/blobs
NITRO_CLI_ARTIFACTS ?= /tmp/nitro-cli-artifacts
HTTP_ADDR ?= :8080

.PHONY: hooks-install go-fmt go-fmt-check go-lint go-check nitro-root-cert ec2-server-run enclave-docker-build enclave-eif-build enclave-run enclave-stop enclave-describe infra-init infra-fmt infra-fmt-check infra-validate infra-check infra-plan infra-deploy infra-destroy infra-state

hooks-install:
	git config core.hooksPath .githooks

go-fmt:
	@for module in $(GO_MODULES); do \
		echo "golangci-lint fmt $$module"; \
		(cd $$module && golangci-lint fmt --config ../.golangci.yml ./...) || exit 1; \
	done

go-fmt-check:
	@for module in $(GO_MODULES); do \
		echo "golangci-lint fmt --diff $$module"; \
		(cd $$module && golangci-lint fmt --config ../.golangci.yml --diff ./...) || exit 1; \
	done

go-lint:
	@for module in $(GO_MODULES); do \
		if find $$module -name '*.go' -type f | grep -q .; then \
			echo "golangci-lint run $$module"; \
			(cd $$module && golangci-lint run --config ../.golangci.yml ./...) || exit 1; \
		else \
			echo "golangci-lint run $$module skipped: no Go files"; \
		fi; \
	done

go-check: go-fmt-check go-lint

nitro-root-cert:
	mkdir -p $(AWS_NITRO_ROOT_CERT_DIR)
	curl -fsSL https://aws-nitro-enclaves.amazonaws.com/AWS_NitroEnclaves_Root-G1.zip -o $(AWS_NITRO_ROOT_CERT_DIR)/AWS_NitroEnclaves_Root-G1.zip
	unzip -p $(AWS_NITRO_ROOT_CERT_DIR)/AWS_NitroEnclaves_Root-G1.zip $(AWS_NITRO_ROOT_CERT_ZIP_ENTRY) > $(AWS_NITRO_ROOT_CERT)
	shasum -a 256 $(AWS_NITRO_ROOT_CERT)

ec2-server-run:
	cd ec2 && ENCLAVE_CID=$(ENCLAVE_CID) ENCLAVE_PORT=$(ENCLAVE_PORT) HTTP_ADDR=$(HTTP_ADDR) go run ./cmd/server

enclave-docker-build:
	docker build -f nitro-enclave/Dockerfile -t $(ENCLAVE_IMAGE) .

enclave-eif-build:
	mkdir -p $(NITRO_CLI_ARTIFACTS)
	NITRO_CLI_BLOBS=$(NITRO_CLI_BLOBS) NITRO_CLI_ARTIFACTS=$(NITRO_CLI_ARTIFACTS) nitro-cli build-enclave --docker-uri $(ENCLAVE_IMAGE) --output-file $(ENCLAVE_EIF)

enclave-run:
	nitro-cli run-enclave --enclave-name $(ENCLAVE_NAME) --eif-path $(ENCLAVE_EIF) --memory $(ENCLAVE_MEMORY_MIB) --cpu-count $(ENCLAVE_CPU_COUNT) --enclave-cid $(ENCLAVE_CID)

enclave-stop:
	nitro-cli terminate-enclave --enclave-name $(ENCLAVE_NAME)

enclave-describe:
	nitro-cli describe-enclaves

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
