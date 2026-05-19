#!/bin/bash

set -e

SERVICE_NAME="webitel-storage"

USER_NAME="webitel"
GROUP_NAME="webitel"

create_user() {
    if ! getent group "$GROUP_NAME" >/dev/null 2>&1; then
        echo "Creating group: $GROUP_NAME"
        addgroup --system "$GROUP_NAME"
    fi

    if ! getent passwd "$USER_NAME" >/dev/null 2>&1; then
        echo "Creating user: $USER_NAME"
        adduser --system --no-create-home --ingroup "$GROUP_NAME" \
                --disabled-password --disabled-login \
                --shell /bin/false \
                --gecos "$PACKAGE_NAME service user" \
                "$USER_NAME"
    fi
}

configure_systemd() {
    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME.service"

    echo "Service $SERVICE_NAME has been installed and enabled."
}

handle_service_restart() {
    echo "Restarting $SERVICE_NAME..."
    systemctl restart "$SERVICE_NAME" || true
}

if [ "$1" == "configure" ]; then
    echo "Configuring $SERVICE_NAME..."

    if [ -z "$2" ]; then
        create_user
        configure_systemd

        echo "$SERVICE_NAME installation completed successfully!"
        echo ""
        echo "Next steps:"
        echo "1. Check status: sudo systemctl status $SERVICE_NAME"
        echo "2. View logs: sudo journalctl -u $SERVICE_NAME -f"
    fi

    handle_service_restart
fi
