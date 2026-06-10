#!/bin/bash

SERVICE_NAME="webitel-storage"

# Stop service if running during upgrade
stop_service_on_upgrade() {
    if [ -x "/bin/systemctl" ]; then
        if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
            echo "Stopping $SERVICE_NAME for upgrade..."
            systemctl stop "$SERVICE_NAME" || true
        fi
    fi
}

if [ "$1" == "upgrade" ]; then
    stop_service_on_upgrade
fi

FK=/opt/storage/key.pem
mkdir -p /opt/storage/data /opt/storage/recordings

if [ ! -f "$FK" ]; then
	openssl genrsa -out $FK 512
fi

exit 0
