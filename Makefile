default: install

generate:
	go generate ./...

install:
	go install .

test:
	go test -count=1 -parallel=4 ./...

testacc:
	TF_ACC=1 go test -count=1 -parallel=4 -timeout 10m -v ./...

test-install: install
	terraform -chdir="./examples/provider-install-verification" plan

test-plan: install
	terraform -chdir="./examples/config" plan

test-apply: install
	terraform -chdir="./examples/config" apply -auto-approve

test-destroy: install
	terraform -chdir="./examples/config" apply -destroy -auto-approve

clean-examples-state:
	rm -r ./examples/**/*tfstate*

clean: clean-examples-state
