#!/bin/bash

set -euo pipefail

# TODO: move this to install script?

workDir="/home/runner/work/pi-app-deployer/pi-app-deployer"
envFile="/usr/local/src/.pi-app-deployer-agent.env"

if [[ $(whoami) != "root" ]]; then
  echo "Script must be run as root"
  exit 1
fi

if [[ -z ${HEROKU_API_KEY} ]]; then
  echo "HEROKU_API_KEY env var not set, exiting now"
  exit 1
fi

rm -f ${envFile}
cat <<< "HEROKU_API_KEY=${HEROKU_API_KEY}" > ${envFile}

mv ${workDir}/pi-app-deployer-agent /usr/local/src/
/usr/local/src/pi-app-deployer-agent install \
    --appUser runneradmin \
    --repoName andrewmarklloyd/pi-test \
    --manifestName pi-test-amd64 \
    --envVar MY_CONFIG=testing \
    --logForwarding \
    --herokuApp ${DEPLOYER_APP}

grep "MY_CONFIG\=testing" /usr/local/src/.pi-test-amd64.env >/dev/null
diff test/test-int-appconfigs.yaml /usr/local/src/.pi-app-deployer.config.yaml

sleep 10
journalctl -u pi-app-deployer-agent.service
journalctl -u pi-test-amd64.service
systemctl is-active pi-app-deployer-agent.service
systemctl is-active pi-test-amd64.service

# trigger an update
git config --global user.name "GitHub Actions Bot"
git config --global user.email "<>"
git clone https://github.com/andrewmarklloyd/pi-test.git
cd pi-test
git remote set-url origin https://andrewmarklloyd:${GH_COMMIT_TOKEN}@github.com/andrewmarklloyd/pi-test.git
deployerSHA=$(git rev-parse HEAD)
echo "Test run: ${deployerSHA};${DEPLOYER_APP}"
echo "${deployerSHA};${DEPLOYER_APP}" >> test/integration-trigger.txt
git add .
git commit -m "Pi App Deployer Test Run ${deployerSHA}"
sha=$(git rev-parse HEAD)
git push origin main

echo "Waiting for successful update of service"
i=0
found="false"
while [[ ${found} == "false" ]]; do
  if [[ ${i} -gt 10 ]]; then
    echo "Exceeded max attempts, test failed"
    exit 1
  fi
  out=$(journalctl -u pi-test-amd64.service -n 100)
  if [[ ${out} == *"${sha}"* ]]; then
    found="true"
  fi
  i=$((i+1))
  sleep 10
done

echo "Successfully ran integration tests! Now update this to use Go testing :)"
