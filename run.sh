#!/bin/bash




curl -L -H "Authorization: token ${GH_API_TOKEN}" https://api.github.com/repos/andrewmarklloyd/pi-test/actions/artifacts/167728953/zip

