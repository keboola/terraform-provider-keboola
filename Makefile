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
test-create: install
	terraform -chdir="./examples/config" plan
	terraform -chdir="./examples/config" apply -auto-approve

test-update: install
	terraform -chdir="./examples/config" plan
	terraform -chdir="./examples/config" apply -auto-approve
test-destroy: install
	terraform -chdir="./examples/config" apply -destroy -auto-approve
