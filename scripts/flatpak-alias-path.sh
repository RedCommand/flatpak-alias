# Add PATH for Flatpak aliases
if [ -d "/var/lib/flatpak/aliases" ]; then
    export PATH="/var/lib/flatpak/aliases:$PATH"
fi