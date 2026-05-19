#!/bin/bash

set -e

SERVICE_NAME="webitel-storage"

stop_service() {
    if [ -x "/bin/systemctl" ]; then
        if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
            echo "Stopping $SERVICE_NAME service..."
            systemctl stop "$SERVICE_NAME" || true

            # Wait for service to stop completely
            local timeout=10
            local count=0
            while systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null && [ $count -lt $timeout ]; do
                sleep 1
                count=$((count + 1))
            done

            if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
                echo "Warning: Service $SERVICE_NAME did not stop gracefully within ${timeout}s" >&2

                # Force kill if still running
                systemctl kill --signal=SIGKILL "$SERVICE_NAME" || true
            else
                echo "Service $SERVICE_NAME stopped successfully."
            fi
        fi
    fi
}

disable_service() {
    if [ -x "/bin/systemctl" ]; then
        if systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
            echo "Disabling $SERVICE_NAME service..."

            systemctl disable "$SERVICE_NAME" || true
            systemctl daemon-reload
        fi
    fi
}

case "$1" in
    remove)
        echo "Preparing to remove $SERVICE_NAME..."

        stop_service
        disable_service

        echo "Service $SERVICE_NAME has been stopped and disabled."
        ;;

    deconfigure)
        echo "Deconfiguring $PACKAGE_NAME due to dependency issues..."

        # Stop service but don't disable it
        stop_service
        ;;

    failed-upgrade)
        echo "Upgrade failed, stopping service..."

        stop_service
        ;;

esac

exit 0
