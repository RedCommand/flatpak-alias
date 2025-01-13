#!/bin/bash

set -e

go build -o flatpak-alias.trigger .
sudo cp flatpak-alias.trigger /usr/share/flatpak/triggers/flatpak-alias.trigger
sudo chmod +x /usr/share/flatpak/triggers/flatpak-alias.trigger