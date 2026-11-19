//go:build linux

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type Battery struct {
	Percent                    int
	OnAC, Charging, HasBattery bool
}

func readFile(p string) string { b, _ := os.ReadFile(p); return strings.TrimSpace(string(b)) }

func readBattery() Battery {
	if previewPercent >= 0 {
		return Battery{Percent: previewPercent, HasBattery: true}
	}
	var st Battery
	dirs, _ := filepath.Glob("/sys/class/power_supply/*")
	total, n := 0, 0
	for _, d := range dirs {
		switch readFile(filepath.Join(d, "type")) {
		case "Battery":
			if readFile(filepath.Join(d, "scope")) == "Device" { // mouse/keyboard batteries
				continue
			}
			if v, err := strconv.Atoi(readFile(filepath.Join(d, "capacity"))); err == nil {
				total += v
				n++
			}
			if readFile(filepath.Join(d, "status")) == "Charging" {
				st.Charging = true
			}
		case "Mains", "USB", "USB_C":
			if readFile(filepath.Join(d, "online")) == "1" {
				st.OnAC = true
			}
		}
	}
	if n == 0 {
		return Battery{Percent: 100, OnAC: true}
	}
	st.HasBattery, st.Percent = true, total/n
	return st
}

// Turn the display off. Tries the common desktops: X11, KDE, GNOME (Wayland), wlroots.
func displayOff() error {
	tries := [][]string{
		{"xset", "dpms", "force", "off"},
		{"kscreen-doctor", "--dpms", "off"},
		{"gdbus", "call", "--session", "--dest", "org.gnome.Mutter.DisplayConfig",
			"--object-path", "/org/gnome/Mutter/DisplayConfig",
			"--method", "org.freedesktop.DBus.Properties.Set",
			"org.gnome.Mutter.DisplayConfig", "PowerSaveMode", "<int32 3>"},
		{"wlopm", "--off", "*"},
	}
	for _, t := range tries {
		if _, err := exec.LookPath(t[0]); err != nil {
			continue
		}
		if t[0] == "xset" && os.Getenv("DISPLAY") == "" {
			continue
		}
		if exec.Command(t[0], t[1:]...).Run() == nil {
			return nil
		}
	}
	return errors.New("no supported way to turn the display off (install xset, kscreen-doctor or wlopm)")
}

func ensureAutostart(exe string) error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	p := filepath.Join(dir, "autostart", "brightcontrol.desktop")
	content := "[Desktop Entry]\nType=Application\nName=BrightControl\n" +
		"Comment=Turns the display off when the battery is almost empty\n" +
		"Exec=\"" + exe + "\"\nX-GNOME-Autostart-enabled=true\nNoDisplay=true\n"
	if readFile(p) == strings.TrimSpace(content) {
		return nil
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	return os.WriteFile(p, []byte(content), 0o644)
}

var lockFile *os.File

func singleInstance() bool {
	dir, _ := os.UserCacheDir()
	_ = os.MkdirAll(dir, 0o755)
	f, err := os.OpenFile(filepath.Join(dir, "brightcontrol.lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return true
	}
	if syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		return false
	}
	lockFile = f
	return true
}

func parentAlive(pid int) bool { return pid > 1 && syscall.Kill(pid, 0) == nil }

func trayForeground() colorNRGBA { return colorNRGBA{255, 255, 255, 255} } // panels are dark on most desktops
