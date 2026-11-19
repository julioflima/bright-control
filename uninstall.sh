#!/bin/bash
# curl -fsSL https://raw.githubusercontent.com/julioflima/bright-control/main/uninstall.sh | bash
LABEL="com.julioflima.brightcontrol"
if [ "$(uname -s)" = "Linux" ]; then
  pkill -x brightcontrol 2>/dev/null || true
  rm -f "$HOME/.local/bin/brightcontrol" "${XDG_CONFIG_HOME:-$HOME/.config}/autostart/brightcontrol.desktop"
  rm -rf "${XDG_CONFIG_HOME:-$HOME/.config}/BrightControl"
  echo "🗑  BrightControl removido."; exit 0
fi
launchctl bootout "gui/$(id -u)/$LABEL" 2>/dev/null || true
pkill -x BrightControl 2>/dev/null || true
rm -f "$HOME/Library/LaunchAgents/$LABEL.plist"
rm -rf /Applications/BrightControl.app "$HOME/Applications/BrightControl.app"
defaults delete "$LABEL" 2>/dev/null || true
echo "🗑  BrightControl removido."
