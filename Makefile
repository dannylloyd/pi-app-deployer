.PHONY: build

build:
	GOOS=linux GOARCH=arm GOARM=5 go build -o pi-app-updater

deploy-dev: build
	scp pi-app-updater pi@${IP}:dev-pi-app-updater
