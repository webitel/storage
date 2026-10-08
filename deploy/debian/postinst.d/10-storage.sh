#!/bin/bash
#
# Storage-specific postinst setup.
#
# Run by the generic Debian postinst (webitel/reusable-configs) BEFORE any
# unit is started — it is sourced under `set -e`, so a failure here aborts
# the install (correct: the service cannot run without its signing key).
#
# Creates the storage working directories and the presigned-URL signing key.

# Create the directories only when missing, so we never change the ownership
# of an existing deployment's data/recordings on upgrade; only files of the
# legacy "webitel" service account are handed over to $USER_NAME.
for dir in /opt/storage /opt/storage/data /opt/storage/recordings; do
    [ -d "$dir" ] || install -d -o "$USER_NAME" -g "$GROUP_NAME" -m 0750 "$dir"
done

I18N_DIR=/usr/share/webitel/storage/i18n
[ -d "$I18N_DIR" ] || install -d -o "$USER_NAME" -g "$GROUP_NAME" -m 0755 "$I18N_DIR"

migrate_legacy_owner /opt/storage

KEY=/opt/storage/key.pem
if [ ! -f "$KEY" ]; then
    echo "Generating storage signing key: $KEY"
    openssl genrsa -out "$KEY" 2048
    chown "$USER_NAME:$GROUP_NAME" "$KEY"
    chmod 600 "$KEY"
fi
