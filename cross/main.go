//go:build linux || windows

// BrightControl for Windows and Linux (the macOS version is the native Swift app
// in ../Sources). Same idea: turn the display off when the battery is almost
// empty and the charger is not connected, with a tray icon to toggle/pause it and
// an extra red battery icon while the battery is low.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"fyne.io/systray"
)

const lowIconThreshold = 20 // low-battery icon shows at this level or below (on battery)

// ---------- settings ----------

type Config struct {
	Enabled     bool `json:"enabled"`
	Threshold   int  `json:"threshold"`
	ShowLowIcon bool `json:"showLowIcon"`
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "BrightControl", "config.json")
}

func loadConfig() Config {
	c := Config{Enabled: true, Threshold: 5, ShowLowIcon: true}
	if b, err := os.ReadFile(configPath()); err == nil {
		_ = json.Unmarshal(b, &c)
	}
	return c
}

func (c Config) save() {
	p := configPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	b, _ := json.MarshalIndent(c, "", "  ")
	_ = os.WriteFile(p, b, 0o644)
}

// ---------- app ----------

type App struct {
	mu          sync.Mutex
	cfg         Config
	pausedUntil time.Time
	lastSleep   time.Time
	indicator   *exec.Cmd

	mStatus, mEnabled, mThreshold, mPause, mResume, mLowIcon *systray.MenuItem
	thresholdItems                                           map[int]*systray.MenuItem
}

func (a *App) paused() bool { return time.Now().Before(a.pausedUntil) }
func (a *App) active() bool { return a.cfg.Enabled && !a.paused() }

func (a *App) onReady() {
	systray.SetTooltip("BrightControl")

	a.mStatus = systray.AddMenuItem("Bateria: …", "")
	a.mStatus.Disable()
	systray.AddSeparator()

	a.mEnabled = systray.AddMenuItemCheckbox("Apagar tela com bateria baixa", "", a.cfg.Enabled)
	a.mThreshold = systray.AddMenuItem("", "")
	a.thresholdItems = map[int]*systray.MenuItem{}
	for _, v := range []int{2, 3, 5, 10, 15, 20} {
		it := a.mThreshold.AddSubMenuItemCheckbox(fmt.Sprintf("%d%%", v), "", v == a.cfg.Threshold)
		a.thresholdItems[v] = it
		go func(v int, it *systray.MenuItem) {
			for range it.ClickedCh {
				a.mu.Lock()
				a.cfg.Threshold = v
				a.cfg.save()
				a.mu.Unlock()
				a.refresh()
			}
		}(v, it)
	}

	a.mPause = systray.AddMenuItem("Pausar por…", "")
	for _, p := range []struct {
		title string
		mins  int
	}{{"15 minutos", 15}, {"30 minutos", 30}, {"1 hora", 60}, {"2 horas", 120}} {
		it := a.mPause.AddSubMenuItem(p.title, "")
		go func(mins int, it *systray.MenuItem) {
			for range it.ClickedCh {
				a.mu.Lock()
				a.pausedUntil = time.Now().Add(time.Duration(mins) * time.Minute)
				a.mu.Unlock()
				a.refresh()
			}
		}(p.mins, it)
	}
	a.mResume = systray.AddMenuItem("Retomar agora", "")
	a.mResume.Hide()

	systray.AddSeparator()
	a.mLowIcon = systray.AddMenuItemCheckbox("Ícone de bateria baixa na barra", "", a.cfg.ShowLowIcon)
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Sair do BrightControl", "")

	go func() {
		for {
			select {
			case <-a.mEnabled.ClickedCh:
				a.mu.Lock()
				a.cfg.Enabled = !a.cfg.Enabled
				a.pausedUntil = time.Time{}
				a.cfg.save()
				a.mu.Unlock()
				a.refresh()
			case <-a.mResume.ClickedCh:
				a.mu.Lock()
				a.pausedUntil = time.Time{}
				a.mu.Unlock()
				a.refresh()
			case <-a.mLowIcon.ClickedCh:
				a.mu.Lock()
				a.cfg.ShowLowIcon = !a.cfg.ShowLowIcon
				a.cfg.save()
				a.mu.Unlock()
				a.refresh()
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()

	a.refresh()
	go func() {
		for range time.Tick(5 * time.Second) {
			a.tick()
		}
	}()
}

func (a *App) onExit() { a.stopIndicator() }

// Runs every 5 s.
func (a *App) tick() {
	a.mu.Lock()
	st := readBattery()
	if st.HasBattery && !st.OnAC && st.Percent <= a.cfg.Threshold && a.active() &&
		time.Since(a.lastSleep) > 20*time.Second && previewPercent < 0 {
		// 20 s grace after each sleep so you can wake the screen and click the icon.
		a.lastSleep = time.Now()
		go displayOff()
	}
	a.mu.Unlock()
	a.refresh()
}

func (a *App) refresh() {
	a.mu.Lock()
	defer a.mu.Unlock()
	st := readBattery()

	switch {
	case !st.HasBattery:
		a.mStatus.SetTitle("Sem bateria")
	case st.Charging:
		a.mStatus.SetTitle(fmt.Sprintf("Bateria: %d%% · carregando", st.Percent))
	case st.OnAC:
		a.mStatus.SetTitle(fmt.Sprintf("Bateria: %d%% · na tomada", st.Percent))
	default:
		a.mStatus.SetTitle(fmt.Sprintf("Bateria: %d%%", st.Percent))
	}

	if a.cfg.Enabled {
		a.mEnabled.Check()
	} else {
		a.mEnabled.Uncheck()
	}
	a.mThreshold.SetTitle(fmt.Sprintf("Apagar quando chegar a %d%%", a.cfg.Threshold))
	for v, it := range a.thresholdItems {
		if v == a.cfg.Threshold {
			it.Check()
		} else {
			it.Uncheck()
		}
	}
	if a.paused() {
		a.mPause.Hide()
		a.mResume.SetTitle("Retomar agora (pausado até " + a.pausedUntil.Format("15:04") + ")")
		a.mResume.Show()
	} else {
		a.mResume.Hide()
		a.mPause.Show()
		if a.cfg.Enabled {
			a.mPause.Enable()
		} else {
			a.mPause.Disable()
		}
	}
	if a.cfg.ShowLowIcon {
		a.mLowIcon.Check()
	} else {
		a.mLowIcon.Uncheck()
	}

	setTrayIcon(moonIcon(a.active()))

	showLow := a.cfg.ShowLowIcon && st.HasBattery && !st.OnAC && st.Percent <= lowIconThreshold
	if showLow {
		a.startIndicator()
	} else {
		a.stopIndicator()
	}
}

// The low-battery icon is a second tray icon; the tray library supports one icon
// per process, so it runs as a child process with no menu (display only).
func (a *App) startIndicator() {
	if a.indicator != nil && a.indicator.ProcessState == nil {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	args := []string{"--battery-indicator"}
	if previewPercent >= 0 {
		args = append(args, "--preview-low", fmt.Sprint(previewPercent))
	}
	cmd := exec.Command(exe, args...)
	if err := cmd.Start(); err != nil {
		return
	}
	a.indicator = cmd
	go func() { _ = cmd.Wait() }()
}

func (a *App) stopIndicator() {
	if a.indicator != nil && a.indicator.Process != nil {
		_ = a.indicator.Process.Kill()
	}
	a.indicator = nil
}

// ---------- low-battery indicator process ----------

func runIndicator() {
	parent := os.Getppid()
	systray.Run(func() {
		update := func() bool {
			st := readBattery()
			if !st.HasBattery || st.OnAC || st.Percent > lowIconThreshold || !parentAlive(parent) {
				return false
			}
			setTrayIcon(batteryIcon(st.Percent))
			systray.SetTitle(fmt.Sprintf("%d%%", st.Percent)) // shown next to the icon where supported (Linux)
			systray.SetTooltip(fmt.Sprintf("Bateria baixa: %d%%", st.Percent))
			return true
		}
		if !update() {
			systray.Quit()
			return
		}
		go func() {
			for range time.Tick(5 * time.Second) {
				if !update() {
					systray.Quit()
					return
				}
			}
		}()
	}, func() {})
}

// ---------- entry ----------

var previewPercent = -1 // --preview-low N: pretend to be on battery at N % (never sleeps the display)

func main() {
	indicator := false
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--battery-indicator":
			indicator = true
		case "--preview-low":
			if i+1 < len(os.Args) {
				fmt.Sscan(os.Args[i+1], &previewPercent)
				i++
			}
		case "--status":
			st := readBattery()
			fmt.Printf("battery=%v percent=%d onAC=%v charging=%v\n", st.HasBattery, st.Percent, st.OnAC, st.Charging)
			return
		case "--display-off":
			if err := displayOff(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}
	}
	if indicator {
		runIndicator()
		return
	}

	if !singleInstance() {
		return
	}
	// Starting at login is mandatory.
	if exe, err := os.Executable(); err == nil {
		_ = ensureAutostart(exe)
	}

	app := &App{cfg: loadConfig()}
	systray.Run(app.onReady, app.onExit)
}
