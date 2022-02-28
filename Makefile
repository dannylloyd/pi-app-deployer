.PHONY: build test

build:
	GOARCH=arm64 GOARM=5 go build -o pi-app-deployer-server server/* 
	GOOS=linux GOARCH=arm GOARM=5 go build -o pi-app-deployer-agent agent/* 
test:
	go test -v ./...

deploy-dev: build
	scp pi-app-deployer-agent pi@${IP}:dev-pi-app-deployer
