FROM alpine

COPY bin/pi-app-deployer-server /app/

WORKDIR /app

ENTRYPOINT ["/app/pi-app-deployer-server"]
