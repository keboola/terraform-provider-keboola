default: install

generate-docs:
	go generate ./...

install:
	go install .

lint:
	golangci-lint run

test:
	go test -count=1 -parallel=4 ./...

testacc:
	TF_ACC=1 go test -count=1 -parallel=1 -timeout 10m -v ./...

test-install: install
	terraform -chdir="./examples/provider-install-verification" plan

test-config-plan: install
	terraform -chdir="./examples/resources/keboola_component_configuration" plan

test-config-apply: install
	terraform -chdir="./examples/resources/keboola_component_configuration" apply -auto-approve

test-config-destroy: install
	terraform -chdir="./examples/resources/keboola_component_configuration" apply -destroy -auto-approve
test-config-show-state: install
	terraform -chdir="./examples/resources/keboola_component_configuration" state show keboola_component_configuration.ex_generic_test
test-config: test-config-destroy test-config-apply test-config-apply test-config-show-state

clean-examples-state:
	rm -r ./examples/**/**/*tfstate*

clean: clean-examples-state
