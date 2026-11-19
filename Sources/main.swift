// BrightControl — menu bar app for macOS.
//
// • Turns the display off when the battery is almost empty (and the charger
//   is not connected), so the remaining charge lasts longer.
// • An icon in the menu bar lets you enable/disable that behaviour, change the
//   threshold or pause it, without opening System Settings.
// • When the battery is low it shows an extra battery icon in the menu bar,
//   drawn like the native macOS one (red fill + percentage).

import Cocoa
import IOKit.ps

// MARK: - Battery

struct BatteryState {
    var percent: Int
    var onAC: Bool
    var charging: Bool
    var hasBattery: Bool

    static func read() -> BatteryState {
        if let p = previewPercent { return BatteryState(percent: p, onAC: false, charging: false, hasBattery: true) }
        let blob = IOPSCopyPowerSourcesInfo().takeRetainedValue()
        let list = IOPSCopyPowerSourcesList(blob).takeRetainedValue() as Array
        let providing = IOPSGetProvidingPowerSourceType(blob)?.takeUnretainedValue() as String?
        let onAC = providing == kIOPSACPowerValue

        for ps in list {
            guard let desc = IOPSGetPowerSourceDescription(blob, ps)?.takeUnretainedValue() as? [String: Any],
                  (desc[kIOPSTypeKey] as? String) == kIOPSInternalBatteryType
            else { continue }
            let cur = desc[kIOPSCurrentCapacityKey] as? Int ?? 0
            let max = desc[kIOPSMaxCapacityKey] as? Int ?? 100
            let pct = max > 0 ? Int((Double(cur) / Double(max) * 100).rounded()) : cur
            let charging = desc[kIOPSIsChargingKey] as? Bool ?? false
            return BatteryState(percent: pct, onAC: onAC, charging: charging, hasBattery: true)
        }
        return BatteryState(percent: 100, onAC: true, charging: false, hasBattery: false)
    }
}

// MARK: - Settings

enum Keys {
    static let enabled = "enabled"
    static let threshold = "threshold"
}

let defaults = UserDefaults.standard

/// The low-battery icon appears at this level or below (on battery).
let lowIconThreshold = 20

/// Set by `--preview-low N`: pretend the Mac is on battery at N % (screenshots/demo; never sleeps the display).
var previewPercent: Int?

// MARK: - Login item (LaunchAgent)

enum LoginItem {
    static let label = "com.julioflima.brightcontrol"
    static var plistURL: URL {
        FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent("Library/LaunchAgents/\(label).plist")
    }

    static func install() {
        if true {
            let exe = Bundle.main.executablePath ?? CommandLine.arguments[0]
            let plist: [String: Any] = [
                "Label": label,
                "ProgramArguments": [exe],
                "RunAtLoad": true,
                // Restart only if it crashes; "Quit" from the menu stays quit.
                "KeepAlive": ["SuccessfulExit": false],
                "ProcessType": "Interactive",
            ]
            try? FileManager.default.createDirectory(at: plistURL.deletingLastPathComponent(),
                                                     withIntermediateDirectories: true)
            if NSDictionary(contentsOf: plistURL) != plist as NSDictionary {
                (plist as NSDictionary).write(to: plistURL, atomically: true)
            }
        }
    }
}

// MARK: - Native-looking battery icon

func batteryImage(percent: Int, low: Bool) -> NSImage {
    let size = NSSize(width: 27, height: 13)
    let img = NSImage(size: size, flipped: false) { _ in
        let body = NSRect(x: 0.5, y: 0.5, width: 23.5, height: 12)
        let outline = NSBezierPath(roundedRect: body, xRadius: 3.6, yRadius: 3.6)
        outline.lineWidth = 1
        NSColor.labelColor.withAlphaComponent(0.55).setStroke()
        outline.stroke()

        // Terminal cap
        let cap = NSBezierPath()
        cap.move(to: NSPoint(x: 25.2, y: 4.5))
        cap.curve(to: NSPoint(x: 25.2, y: 8.5),
                  controlPoint1: NSPoint(x: 26.8, y: 5.2),
                  controlPoint2: NSPoint(x: 26.8, y: 7.8))
        cap.close()
        NSColor.labelColor.withAlphaComponent(0.55).setFill()
        cap.fill()

        // Charge level
        let inner = body.insetBy(dx: 2, dy: 2)
        let frac = max(0.08, min(1, CGFloat(percent) / 100))
        let fill = NSRect(x: inner.minX, y: inner.minY, width: inner.width * frac, height: inner.height)
        (low ? NSColor.systemRed : NSColor.labelColor).setFill()
        NSBezierPath(roundedRect: fill, xRadius: 1.8, yRadius: 1.8).fill()
        return true
    }
    img.isTemplate = false
    return img
}

/// Transparent view that eats mouse clicks (the low-battery icon is display-only).
final class ClickShield: NSView {
    override func mouseDown(with event: NSEvent) {}
    override func mouseUp(with event: NSEvent) {}
    override func rightMouseDown(with event: NSEvent) {}
    override func otherMouseDown(with event: NSEvent) {}
    override func acceptsFirstMouse(for event: NSEvent?) -> Bool { true }
}

// MARK: - App

/// New status items are added on the far left and, on MacBooks with a notch and a busy
/// menu bar, end up hidden behind it. Seed a preferred position near the clock (points
/// from the right edge) the first time; after that macOS remembers wherever you ⌘-drag it.
func makeStatusItem(_ name: String, length: CGFloat, position: Double) -> NSStatusItem {
    let key = "NSStatusItem Preferred Position \(name)"
    if defaults.object(forKey: key) == nil { defaults.set(position, forKey: key) }
    let item = NSStatusBar.system.statusItem(withLength: length)
    item.autosaveName = name
    return item
}

final class AppDelegate: NSObject, NSApplicationDelegate, NSMenuDelegate {
    let controlItem = makeStatusItem("BrightControl", length: NSStatusItem.squareLength, position: 300)
    var batteryItem: NSStatusItem?
    var timer: Timer?
    var pausedUntil: Date?
    var lastSleep = Date.distantPast
    var state = BatteryState.read()

    func applicationDidFinishLaunching(_ note: Notification) {
        defaults.register(defaults: [
            Keys.enabled: true,
            Keys.threshold: 5,
        ])

        let menu = NSMenu()
        menu.delegate = self
        controlItem.menu = menu
        controlItem.button?.toolTip = "BrightControl"

        // Starting at login is mandatory: make sure the LaunchAgent exists on every
        // launch (only for an installed copy, never for a dev build).
        let path = Bundle.main.bundlePath
        if path.hasPrefix("/Applications/") || path.hasPrefix(NSHomeDirectory() + "/Applications/") {
            LoginItem.install()
        }

        tick()
        timer = Timer.scheduledTimer(withTimeInterval: 5, repeats: true) { [weak self] _ in self?.tick() }
        timer?.tolerance = 1

        // Debug helper: `--open-menu` pops the menu open (for screenshots).
        if CommandLine.arguments.contains("--open-menu") {
            DispatchQueue.main.asyncAfter(deadline: .now() + 1) { self.controlItem.button?.performClick(nil) }
        }
    }

    var isPaused: Bool { (pausedUntil.map { $0 > Date() }) ?? false }
    var isActive: Bool { defaults.bool(forKey: Keys.enabled) && !isPaused }

    // Runs every 5 s.
    func tick() {
        state = BatteryState.read()
        if let until = pausedUntil, until <= Date() { pausedUntil = nil }

        let threshold = defaults.integer(forKey: Keys.threshold)
        if isActive, previewPercent == nil, state.hasBattery, !state.onAC, state.percent <= threshold,
           Date().timeIntervalSince(lastSleep) > 20 {
            // 20 s grace after each sleep so you can wake the screen and click the icon.
            lastSleep = Date()
            let p = Process()
            p.executableURL = URL(fileURLWithPath: "/usr/bin/pmset")
            p.arguments = ["displaysleepnow"]
            try? p.run()
        }
        refreshIcons()
    }

    func refreshIcons() {
        // Control icon
        let name = isActive ? "moon.zzz.fill" : "moon.zzz"
        let img = NSImage(systemSymbolName: name, accessibilityDescription: "BrightControl")
        img?.isTemplate = true
        controlItem.button?.image = img
        controlItem.button?.appearsDisabled = !isActive

        // Low battery icon
        let showLow = state.hasBattery
            && !state.onAC && state.percent <= lowIconThreshold
        if showLow {
            if batteryItem == nil {
                batteryItem = makeStatusItem("BrightControlBattery", length: NSStatusItem.variableLength, position: 330)
                batteryItem?.button?.imagePosition = .imageRight
                batteryItem?.button?.font = NSFont.menuBarFont(ofSize: 0)
                if let button = batteryItem?.button {
                    // Indicator only: swallow clicks so the item is never highlighted/selected.
                    let shield = ClickShield(frame: button.bounds)
                    shield.autoresizingMask = [.width, .height]
                    button.addSubview(shield)
                }
            }
            batteryItem?.button?.title = "\(state.percent)%"
            batteryItem?.button?.image = batteryImage(percent: state.percent, low: true)
            batteryItem?.button?.toolTip = "Bateria baixa: \(state.percent)%"
        } else if let item = batteryItem {
            NSStatusBar.system.removeStatusItem(item)
            batteryItem = nil
        }
    }

    // MARK: Menu

    func menuNeedsUpdate(_ menu: NSMenu) {
        state = BatteryState.read()
        menu.removeAllItems()

        let status: String
        if !state.hasBattery { status = "Sem bateria" }
        else if state.charging { status = "Bateria: \(state.percent)% · carregando" }
        else if state.onAC { status = "Bateria: \(state.percent)% · na tomada" }
        else { status = "Bateria: \(state.percent)%" }
        let header = NSMenuItem(title: status, action: nil, keyEquivalent: "")
        header.isEnabled = false
        menu.addItem(header)
        menu.addItem(.separator())

        let threshold = defaults.integer(forKey: Keys.threshold)
        let enabled = item("Apagar tela com bateria baixa", #selector(toggleEnabled))
        enabled.state = defaults.bool(forKey: Keys.enabled) ? .on : .off
        menu.addItem(enabled)

        let thMenu = NSMenu()
        for v in [2, 3, 5, 10, 15, 20] {
            let i = item("\(v)%", #selector(setThreshold(_:)))
            i.tag = v
            i.state = v == threshold ? .on : .off
            thMenu.addItem(i)
        }
        let th = NSMenuItem(title: "Apagar quando chegar a \(threshold)%", action: nil, keyEquivalent: "")
        th.submenu = thMenu
        menu.addItem(th)

        if isPaused, let until = pausedUntil {
            let f = DateFormatter(); f.timeStyle = .short
            menu.addItem(item("Retomar agora (pausado até \(f.string(from: until)))", #selector(resume)))
        } else {
            let pMenu = NSMenu()
            for (title, mins) in [("15 minutos", 15), ("30 minutos", 30), ("1 hora", 60), ("2 horas", 120)] {
                let i = item(title, #selector(pause(_:)))
                i.tag = mins
                pMenu.addItem(i)
            }
            let p = NSMenuItem(title: "Pausar por…", action: nil, keyEquivalent: "")
            p.submenu = pMenu
            p.isEnabled = defaults.bool(forKey: Keys.enabled)
            menu.addItem(p)
        }


        menu.addItem(.separator())
        menu.addItem(item("Sair do BrightControl", #selector(quit), key: "q"))
    }

    func item(_ title: String, _ action: Selector, key: String = "") -> NSMenuItem {
        let i = NSMenuItem(title: title, action: action, keyEquivalent: key)
        i.target = self
        return i
    }

    @objc func toggleEnabled() {
        defaults.set(!defaults.bool(forKey: Keys.enabled), forKey: Keys.enabled)
        pausedUntil = nil
        refreshIcons()
    }
    @objc func setThreshold(_ s: NSMenuItem) { defaults.set(s.tag, forKey: Keys.threshold); tick() }
    @objc func pause(_ s: NSMenuItem) { pausedUntil = Date().addingTimeInterval(Double(s.tag) * 60); refreshIcons() }
    @objc func resume() { pausedUntil = nil; tick() }
    @objc func quit() { NSApp.terminate(nil) }
}

// Debug helper: `BrightControl --preview-low 8` forces the low-battery icon (for screenshots).
if let i = CommandLine.arguments.firstIndex(of: "--preview-low"), i + 1 < CommandLine.arguments.count,
   let pct = Int(CommandLine.arguments[i + 1]) {
    previewPercent = pct
}

let app = NSApplication.shared
let delegate = AppDelegate()
app.delegate = delegate
app.setActivationPolicy(.accessory)
app.run()
