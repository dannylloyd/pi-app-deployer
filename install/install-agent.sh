#!/bin/bash

set -euo pipefail

get_latest_release() {
  curl --silent "https://api.github.com/repos/andrewmarklloyd/pi-app-updater/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

if [[ $(whoami) != "root" ]]; then
  echo "Script must be run as root"
  exit 1
fi

if [[ -z ${HEROKU_API_KEY} ]]; then
  echo "HEROKU_API_KEY env var not set, exiting now"
  exit 1
fi

if ! command -v jq &> /dev/null; then
  sudo apt-get update
  sudo apt-get install jq -y
fi

vars=$(curl -s -n https://api.heroku.com/apps/pi-app-updater/config-vars \
  -H "Accept: application/vnd.heroku+json; version=3" \
  -H "Authorization: Bearer ${HEROKU_API_KEY}")

reqVars="GH_API_TOKEN
PI_APP_UPDATER_API_KEY
CLOUDMQTT_AGENT_USER
CLOUDMQTT_AGENT_PASSWORD
CLOUDMQTT_URL"

envFile="/home/pi/.pi-app-updater-agent.env"
rm -f ${envFile}
for key in ${reqVars}; do
  val=$(echo $vars | jq -r ".${key}")
  export "${key}=${val}"
  echo "${key}=${val}" >> ${envFile}
done

version=$(get_latest_release)
curl -sL https://github.com/andrewmarklloyd/pi-app-updater/releases/download/${version}/pi-app-updater-agent-${version}-linux-arm.tar.gz | tar zx -C /tmp

mv /tmp/pi-app-updater-agent /home/pi/pi-app-updater-agent

echo "Enter the repo name including the org then press enter:"
read repo

echo "Enter the package name included in the artifact then press enter:"
read package

echo
echo "Press enter to run the pi-app-updater installer using the following command:"

c="/home/pi/pi-app-updater-agent --repo-name ${repo} --package-name ${package} --install"
echo "${c}"
read

eval ${c}
