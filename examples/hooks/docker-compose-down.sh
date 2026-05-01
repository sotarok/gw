#!/bin/bash
# Tear down docker compose containers and volumes for the worktree before
# gw removes it. Runs with cwd set to the worktree, so `docker compose` finds
# the compose file naturally.
#
# Usage in ~/.gwrc:
#   pre_end_hook = ~/.gw/hooks/docker-compose-down.sh

set -euo pipefail

echo "[gw pre_end_hook] $GW_COMMAND for $GW_BRANCH_NAME @ $GW_WORKTREE_PATH"

# Collect compose files in the worktree root, including variants like
# docker-compose.dev.yml, compose.override.yaml, etc.
shopt -s nullglob
compose_args=()
for f in docker-compose.yml docker-compose.yaml compose.yml compose.yaml \
         docker-compose.*.yml docker-compose.*.yaml \
         compose.*.yml compose.*.yaml; do
  [ -f "$f" ] && compose_args+=( -f "$f" )
done
shopt -u nullglob

if [ ${#compose_args[@]} -eq 0 ]; then
  echo "[gw pre_end_hook] no docker compose file found, skipping"
  exit 0
fi

echo "[gw pre_end_hook] docker compose ${compose_args[*]} down -v --remove-orphans"
docker compose "${compose_args[@]}" down -v --remove-orphans
