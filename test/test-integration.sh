#!/bin/bash

set -euo pipefail

# TODO: move this to install script?

os="Ubuntu"
workDir="/home/runner/work/pi-app-deployer/pi-app-deployer"
homeDir="/home/runner"
envFile="${homeDir}/.pi-app-deployer-agent.env"

if [[ $(whoami) != "root" ]]; then
  echo "Script must be run as root"
  exit 1
fi

if [[ -z ${HEROKU_API_KEY} ]]; then
  echo "HEROKU_API_KEY env var not set, exiting now"
  exit 1
fi

if ! command -v jq &> /dev/null; then
  apt-get update
  apt-get install jq -y
fi

if ! command -v curl &> /dev/null; then
  apt-get update
  apt-get install curl -y
fi

vars=$(curl -s -n https://api.heroku.com/apps/pi-app-deployer/config-vars \
  -H "Accept: application/vnd.heroku+json; version=3" \
  -H "Authorization: Bearer ${HEROKU_API_KEY}")

reqVars="GH_API_TOKEN
PI_APP_DEPLOYER_API_KEY
CLOUDMQTT_AGENT_USER
CLOUDMQTT_AGENT_PASSWORD
CLOUDMQTT_URL"

rm -f ${envFile}
echo "HEROKU_API_KEY=${HEROKU_API_KEY}" > ${envFile}
for key in ${reqVars}; do
  val=$(echo $vars | jq -r ".${key}")
  export "${key}=${val}"
  echo "${key}=${val}" >> ${envFile}
done

mv ${workDir}/pi-app-deployer-agent ${homeDir}
${homeDir}/pi-app-deployer-agent --app-user runneradmin --repo-name ${repo} --manifest-name ${manifestName} --install

sleep 10
systemctl is-active --quiet pi-app-deployer-agent.service
systemctl is-active --quiet pi-test.service
