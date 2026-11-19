#!/bin/bash
# BrightControl installer
#   curl -fsSL https://raw.githubusercontent.com/julioflima/bright-control/main/install.sh | bash
set -euo pipefail
REPO="julioflima/bright-control"

# ---------- Linux ----------
if [ "$(uname -s)" = "Linux" ]; then
  case "$(uname -m)" in
    x86_64|amd64) ARCH=amd64 ;;
    aarch64|arm64) ARCH=arm64 ;;
    *) echo "Arquitetura não suportada: $(uname -m)"; exit 1 ;;
  esac
  BIN="$HOME/.local/bin/brightcontrol"
  echo "🔋 Instalando BrightControl em $BIN…"
  mkdir -p "$(dirname "$BIN")"
  pkill -x brightcontrol 2>/dev/null || true
  curl -fsSL "https://github.com/$REPO/releases/latest/download/brightcontrol-linux-$ARCH" -o "$BIN.tmp"
  chmod +x "$BIN.tmp" && mv "$BIN.tmp" "$BIN"
  # The app registers itself in ~/.config/autostart on first run.
  (setsid "$BIN" >/dev/null 2>&1 &) || (nohup "$BIN" >/dev/null 2>&1 &)
  echo "✅ Pronto! Procure o ícone 🌙 na bandeja do sistema."
  echo "   GNOME: precisa da extensão AppIndicator para mostrar ícones na bandeja."
  exit 0
fi

# ---------- macOS ----------
LABEL="com.julioflima.brightcontrol"
AGENT="$HOME/Library/LaunchAgents/$LABEL.plist"

if [ -w /Applications ]; then DEST=/Applications; else DEST="$HOME/Applications"; mkdir -p "$DEST"; fi
APP="$DEST/BrightControl.app"

echo "🔋 Instalando BrightControl em $DEST…"

# Replace the old bright_control.sh LaunchAgent, if present
OLD="$HOME/Library/LaunchAgents/com.usuario.controlbrillo.plist"
if [ -f "$OLD" ]; then
  echo "🧹 Removendo o antigo bright_control.sh (com.usuario.controlbrillo)"
  launchctl bootout "gui/$(id -u)/com.usuario.controlbrillo" 2>/dev/null || true
  rm -f "$OLD"
fi

# Stop a running copy
launchctl bootout "gui/$(id -u)/$LABEL" 2>/dev/null || true
pkill -x BrightControl 2>/dev/null || true

TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
if [ -n "${LOCAL_ZIP:-}" ]; then cp "$LOCAL_ZIP" "$TMP/BrightControl.zip"
else
  echo "⬇️  Baixando a última versão…"
  curl -fsSL "https://github.com/$REPO/releases/latest/download/BrightControl.zip" -o "$TMP/BrightControl.zip"
fi
rm -rf "$APP"
ditto -x -k "$TMP/BrightControl.zip" "$DEST"
xattr -dr com.apple.quarantine "$APP" 2>/dev/null || true

# Start at login
mkdir -p "$(dirname "$AGENT")"
cat > "$AGENT" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>$LABEL</string>
  <key>ProgramArguments</key><array><string>$APP/Contents/MacOS/BrightControl</string></array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>ProcessType</key><string>Interactive</string>
</dict>
</plist>
PLIST
launchctl bootstrap "gui/$(id -u)" "$AGENT"

echo "✅ Pronto! Procure o ícone 🌙 na barra de menus, lá em cima."
