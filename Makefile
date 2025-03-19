default: install

generate-docs:
	go generate ./...

install:
	go install .

lint:
	golangci-lint run -c "./.golangci.yml"

fix:
	@echo "Running go mod tidy ..."
	go mod tidy

	@echo "Running gofumpt ..."
	gofumpt -w ./internal

	@echo "Running gci ..."
	gci write --skip-generated -s standard -s default -s "prefix(github.com/keboola/terraform-provider-keboola)" ./internal

	@echo "Running golangci-lint ..."
	golangci-lint run --fix -c "./.golangci.yml"

test:
	go test -count=1 -parallel=4 ./...

testacc:
	TF_ACC=1 go test -count=1 -parallel=1 -timeout 10m -v ./...

testacc-encrypt:
	TF_ACC=1 go test -count=1 -parallel=1 -timeout 10m -v ./... -run TestAccEncryptionResource

test-install: install
	terraform -chdir="./examples/provider-install-verification" plan

test-encrypt-plan: install
	terraform -chdir="./examples/resources/keboola_encryption" plan

test-encrypt-apply: install
	terraform -chdir="./examples/resources/keboola_encryption" apply -auto-approve

test-encrypt-show: install
	terraform -chdir="./examples/resources/keboola_encryption" state show keboola_encryption.encryption_test

test-encrypt-destroy: install
	terraform -chdir="./examples/resources/keboola_encryption" apply -destroy -auto-approve

test-config-plan: install
	terraform -chdir="./examples/resources/keboola_component_configuration" plan

test-config-apply: install
	terraform -chdir="./examples/resources/keboola_component_configuration" apply -auto-approve

test-config-destroy: install
	terraform -chdir="./examples/resources/keboola_component_configuration" apply -destroy -auto-approve
test-config-show-state: install
	terraform -chdir="./examples/resources/keboola_component_configuration" state show keboola_component_configuration.ex_generic_test

test-config: test-config-destroy test-config-apply test-config-apply test-config-show-state

test-telemetry-init: install
	terraform -chdir="./examples/telemetry" init

test-telemetry-plan: install test-telemetry-init
	TF_LOG=info terraform -chdir="./examples/telemetry" plan

test-telemetry-apply: install test-telemetry-init
	TF_LOG=info terraform -chdir="./examples/telemetry" apply -auto-approve

test-telemetry-destroy: install test-telemetry-init
	terraform -chdir="./examples/telemetry" apply -destroy -auto-approve

test-telemetry-show-state: install
	terraform -chdir="./examples/telemetry" state show keboola_component_configuration_row.telemetry_extractor

test-telemetry: test-telemetry-destroy test-telemetry-apply test-telemetry-show-state

clean-examples-state:
	rm -r ./examples/**/**/*tfstate* || true
	rm -r ./examples/**/**/.terraform.lock.hcl || true
	rm -rf ./examples/**/**/.terraform*

clean: clean-examples-state

	rm -rf ./examples/**/**/.terraform*
