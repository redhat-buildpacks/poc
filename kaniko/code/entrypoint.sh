#!/bin/sh

# https://chaseadams.io/posts/fix-docker-error-exec-user-process-caused-no-such-file-or-directory/

echo "Env var DEBUG: ${DEBUG}"

if [[ -z "${DEBUG}" ]]; then
  /kaniko-app
else
  /usr/local/bin/dlv --listen=:4000 --headless=true --api-version=2 --accept-multiclient exec /kaniko-app
fi