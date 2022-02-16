.PHONY: build test

build:
	GOOS=linux GOARCH=arm GOARM=5 go build -o pi-app-updater

test:
	go test -v ./...

deploy-dev: build
	scp pi-app-updater pi@${IP}:dev-pi-app-updater
