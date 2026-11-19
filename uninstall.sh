#!/bin/bash
# curl -fsSL https://raw.githubusercontent.com/julioflima/bright-control/main/uninstall.sh | bash
LABEL="com.julioflima.brightcontrol"
launchctl bootout "gui/$(id -u)/$LABEL" 2>/dev/null || true
pkill -x BrightControl 2>/dev/null || true
rm -f "$HOME/Library/LaunchAgents/$LABEL.plist"
rm -rf /Applications/BrightControl.app "$HOME/Applications/BrightControl.app"
defaults delete "$LABEL" 2>/dev/null || true
echo "🗑  BrightControl removido."
