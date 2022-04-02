#!/bin/bash

set -euo pipefail

interactive=${interactive:-}

deployerDir="/usr/local/src"
envFile="${deployerDir}/.pi-app-deployer-agent.env"

osRelease=$(cat /etc/os-release)
if [[ "${osRelease}" == *"Raspbian"* ]]; then
  arch="arm"
elif [[ "${osRelease}" == *"Ubuntu"* ]]; then
  arch="amd64"
else
  echo "OS not supported"
  exit 1
fi

get_latest_release() {
  curl --silent "https://api.github.com/repos/andrewmarklloyd/pi-app-deployer/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
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
  apt-get update
  apt-get install jq -y
fi

if ! command -v curl &> /dev/null; then
  apt-get update
  apt-get install curl -y
fi

# TODO: write this file in the install command
rm -f ${envFile}
echo "HEROKU_API_KEY=${HEROKU_API_KEY}" > ${envFile}

version=$(get_latest_release)
echo "Downloading version ${version} of pi-app-deployer"
curl -sL https://github.com/andrewmarklloyd/pi-app-deployer/releases/download/${version}/pi-app-deployer-agent-${version}-linux-${arch}.tar.gz | tar zx -C /tmp

mv /tmp/pi-app-deployer-agent ${deployerDir}/pi-app-deployer-agent

if [[ ${interactive} == "true" ]]; then
  echo "Enter the repo name including the org then press enter:"
  read repo

  echo "Enter the pi-app-deployer manifest name then press enter:"
  read manifestName

  echo
  echo "Running the pi-app-deployer-agent version ${version} installer using the following command:"

  c="${deployerDir}/pi-app-deployer-agent --repo-name ${repo} --manifest-name ${manifestName} --install"
  echo "${c}"

  eval ${c}
else
  echo "pi-app-deployer-agent downloaded, run from ${deployerDir}/pi-app-deployer-agent"
fi
