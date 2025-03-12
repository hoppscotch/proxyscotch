#!/bin/sh

# Default values
DEFAULT_TOKEN=""
DEFAULT_ALLOWED_ORIGINS="*"
DEFAULT_BANNED_OUTPUTS=""
DEFAULT_BANNED_DESTS=""

# Proxyscotch container allows configurations through env variables
# in PROXYSCOTCH_TOKEN, PROXYSCOTCH_ALLOWED_ORIGINS,
# PROXYSCOTCH_BANNED_OUTPUTS and PROXYSCOTCH_BANNED_DESTS

# This is hardcoded
HOST_ARG="--host=0.0.0.0:9159"

# Process token (only add if env var is set or default is not blank)
TOKEN_ARG=""
if [ -n "${PROXYSCOTCH_TOKEN}" ]; then
  TOKEN_ARG="--token=${PROXYSCOTCH_TOKEN}"
elif [ -n "${DEFAULT_TOKEN}" ]; then
  TOKEN_ARG="--token=${DEFAULT_TOKEN}"
fi

# Process allowed-origins
if [ -n "${PROXYSCOTCH_ALLOWED_ORIGINS}" ]; then
  ORIGINS_ARG="--allowed-origins=${PROXYSCOTCH_ALLOWED_ORIGINS}"
else
  ORIGINS_ARG="--allowed-origins=${DEFAULT_ALLOWED_ORIGINS}"
fi

# Process banned-outputs (only add if env var is set or default is not blank)
BANNED_OUTPUTS_ARG=""
if [ -n "${PROXYSCOTCH_BANNED_OUTPUTS}" ]; then
  BANNED_OUTPUTS_ARG="--banned-outputs=${PROXYSCOTCH_BANNED_OUTPUTS}"
elif [ -n "${DEFAULT_BANNED_OUTPUTS}" ]; then
  BANNED_OUTPUTS_ARG="--banned-outputs=${DEFAULT_BANNED_OUTPUTS}"
fi

# Process banned-dests (only add if env var is set or default is not blank)
BANNED_DESTS_ARG=""
if [ -n "${PROXYSCOTCH_BANNED_DESTS}" ]; then
  BANNED_DESTS_ARG="--banned-dests=${PROXYSCOTCH_BANNED_DESTS}"
elif [ -n "${DEFAULT_BANNED_DESTS}" ]; then
  BANNED_DESTS_ARG="--banned-dests=${DEFAULT_BANNED_DESTS}"
fi

# Execute the command with the arguments
proxyscotch $HOST_ARG $TOKEN_ARG $ORIGINS_ARG $BANNED_OUTPUTS_ARG $BANNED_DESTS_ARG
