#!/bin/bash

# Set initial arguments for xcomfortd
set -- xcomfortd \
    --client-id "${MQTT_CLIENT_ID}" \
    --server "tcp://${MQTT_USER}:${MQTT_PASSWORD}@${MQTT_HOST}:${MQTT_PORT}" \
    --hadiscoveryprefix "${HA_DISCOVERY_PREFIX}"

# Add EPROM option if EPROM is true
if [ "$EPROM" = "true" ]; then
    set -- "$@" --eprom
fi

# Add Datapoints file option if it's set
if [ "$DATAPOINTS_FILE" != "" ]; then
    set -- "$@" --file "${CONFIG_PATH}/${DATAPOINTS_FILE}"
fi

# Add discovery options based on HA_DISCOVERY and HA_DISCOVERY_REMOVE
if [ "$HA_DISCOVERY" = "true" ]; then
    set -- "$@" --hadiscovery
fi

if [ "$HA_DISCOVERY_REMOVE" = "true" ]; then
    set -- "$@" --hadiscoveryremove
fi

# Add verbosity option if VERBOSE is true
if [ "$VERBOSE" = "true" ]; then
    set -- "$@" --verbose
fi

# Add ECI_HOSTS as multiple --host arguments
for i in $ECI_HOSTS; do
    set -- "$@" --host "$i"
done

# Add HIDAPI option if HIDAPI is true
if [ "$HIDAPI" = "true" ]; then
    set -- "$@" --hidapi
fi

# Finally, execute the command with all the arguments
exec "$@"
