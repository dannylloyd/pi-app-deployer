.PHONY: build test

build:
	GOARCH=arm64 GOARM=5 go build -o pi-app-updater-server server/* 
	GOOS=linux GOARCH=arm GOARM=5 go build -o pi-app-updater-agent agent/* 
test:
	go test -v ./...

deploy-dev: build
	scp pi-app-updater-agent pi@${IP}:dev-pi-app-updater
