//go:build windows

package main

import (
	"errors"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

type Battery struct {
	Percent                    int
	OnAC, Charging, HasBattery bool
}

var (
	kernel32                 = windows.NewLazySystemDLL("kernel32.dll")
	user32                   = windows.NewLazySystemDLL("user32.dll")
	procGetSystemPowerStatus = kernel32.NewProc("GetSystemPowerStatus")
	procPostMessageW         = user32.NewProc("PostMessageW")
)

type systemPowerStatus struct {
	ACLineStatus        byte
	BatteryFlag         byte
	BatteryLifePercent  byte
	SystemStatusFlag    byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

func readBattery() Battery {
	if previewPercent >= 0 {
		return Battery{Percent: previewPercent, HasBattery: true}
	}
	var s systemPowerStatus
	if r, _, _ := procGetSystemPowerStatus.Call(uintptr(unsafe.Pointer(&s))); r == 0 {
		return Battery{Percent: 100, OnAC: true}
	}
	if s.BatteryFlag&128 != 0 || s.BatteryFlag == 255 || s.BatteryLifePercent == 255 { // no battery / unknown
		return Battery{Percent: 100, OnAC: s.ACLineStatus == 1}
	}
	return Battery{
		Percent:    int(s.BatteryLifePercent),
		OnAC:       s.ACLineStatus == 1,
		Charging:   s.BatteryFlag&8 != 0,
		HasBattery: true,
	}
}

// SC_MONITORPOWER with lParam 2 = power the display off.
func displayOff() error {
	const hwndBroadcast, wmSyscommand, scMonitorpower = 0xFFFF, 0x0112, 0xF170
	r, _, err := procPostMessageW.Call(hwndBroadcast, wmSyscommand, scMonitorpower, 2)
	if r == 0 {
		return err
	}
	return nil
}

func ensureAutostart(exe string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	want := `"` + exe + `"`
	if cur, _, err := k.GetStringValue("BrightControl"); err == nil && strings.EqualFold(cur, want) {
		return nil
	}
	return k.SetStringValue("BrightControl", want)
}

func singleInstance() bool {
	name, _ := windows.UTF16PtrFromString("Local\\BrightControlSingleInstance")
	_, err := windows.CreateMutex(nil, false, name)
	return !errors.Is(err, windows.ERROR_ALREADY_EXISTS)
}

func parentAlive(pid int) bool {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	ev, _ := windows.WaitForSingleObject(h, 0)
	return ev == uint32(windows.WAIT_TIMEOUT)
}

// Light or dark taskbar.
func trayForeground() colorNRGBA {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		if v, _, err := k.GetIntegerValue("SystemUsesLightTheme"); err == nil && v == 1 {
			return colorNRGBA{0, 0, 0, 255}
		}
	}
	return colorNRGBA{255, 255, 255, 255}
}
