FROM alpine

COPY bin/pi-app-deployer-server /app/

RUN chmod +x /app/pi-app-deployer-server

WORKDIR /app

ENTRYPOINT ["/app/pi-app-deployer-server"]
