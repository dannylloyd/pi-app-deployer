.PHONY: build test

GIT_REV=`git rev-parse --short HEAD`
GIT_TREE_STATE=$(shell (git status --porcelain | grep -q .) && echo $(GIT_REV)-dirty || echo $(GIT_REV))

build:
	CGO_ENABLED=0 GOARCH=amd64 go build -ldflags="-X 'main.version=$(GIT_TREE_STATE)'" -o bin/pi-app-deployer-server server/*
	GOOS=linux GOARCH=arm GOARM=5 go build -ldflags="-X 'github.com/andrewmarklloyd/pi-app-deployer/cmd.version=$(GIT_TREE_STATE)'" -o bin/pi-app-deployer-agent agent/main.go

test:
	go test -v ./...

test-integration:
	GOOS=linux GOARCH=amd64 GOARM=5 go build -o pi-app-deployer-agent agent/main.go
	sudo -E ./test/test-integration.sh

deploy-agent: build
	ansible-playbook playbook.yaml

deploy-agent-test: build
	ansible-playbook test-playbook.yaml --extra-vars "variable_host=${HOST}"

clean:
	rm -rf bin/
