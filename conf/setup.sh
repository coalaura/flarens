#!/bin/bash

set -euo pipefail

echo "Linking sysusers config..."

mkdir -p /etc/sysusers.d

if [ ! -f /etc/sysusers.d/flarens.conf ]; then
    ln -s "/opt/flarens/conf/flarens.conf" /etc/sysusers.d/flarens.conf
fi

echo "Creating user..."
systemd-sysusers

echo "Linking unit..."
rm /etc/systemd/system/flarens.service

systemctl link "/opt/flarens/conf/flarens.service"

echo "Reloading daemon..."
systemctl daemon-reload
systemctl enable flarens

echo "Fixing initial permissions..."
chown -R flarens:flarens "/opt/flarens"

find "/opt/flarens" -type d -exec chmod 755 {} +
find "/opt/flarens" -type f -exec chmod 644 {} +

chmod +x "/opt/flarens/flarens"

echo "Setup complete, starting service..."

service flarens start

echo "Done."
