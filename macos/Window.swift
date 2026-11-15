import AppKit

struct GameCell: Decodable {
    let i: Int
    let t: String?
    let w: Int
    let f: Int
    let b: Int
}

struct GameFrame: Decodable {
    let width: Int
    let height: Int
    let background: Int
    let cells: [GameCell]
}

final class GameView: NSView {
    private let cellWidth: CGFloat = 14
    private let cellHeight: CGFloat = 28
    private let padding: CGFloat = 26
    private let font = NSFont.monospacedSystemFont(ofSize: 19, weight: .medium)
    private let wideFont = NSFont.systemFont(ofSize: 23, weight: .medium)
    private var colors: [Int: NSColor] = [:]
    var send: (String) -> Void = { _ in }
    var gameFrame: GameFrame? { didSet { needsDisplay = true } }

    override var isFlipped: Bool { true }
    override var acceptsFirstResponder: Bool { true }

    private func layout(for frame: GameFrame) -> (scale: CGFloat, origin: CGPoint) {
        let width = CGFloat(frame.width) * cellWidth
        let height = CGFloat(frame.height) * cellHeight
        let scale = min((bounds.width - 2 * padding) / width, (bounds.height - 2 * padding) / height)
        return (scale, CGPoint(x: (bounds.width - width * scale) / 2, y: (bounds.height - height * scale) / 2))
    }

    private func color(_ hex: Int) -> NSColor {
        let hex = hex < 0 ? 0xe4dbfb : hex
        if let known = colors[hex] { return known }
        let value = NSColor(srgbRed: CGFloat((hex >> 16) & 255) / 255,
                            green: CGFloat((hex >> 8) & 255) / 255,
                            blue: CGFloat(hex & 255) / 255, alpha: 1)
        colors[hex] = value
        return value
    }

    override func draw(_ dirtyRect: NSRect) {
        guard let context = NSGraphicsContext.current?.cgContext else { return }
        context.setFillColor(color(gameFrame?.background ?? 0x101222).cgColor)
        context.fill(bounds)
        guard let frame = gameFrame else { return }
        let placement = layout(for: frame)
        context.saveGState()
        defer { context.restoreGState() }
        context.translateBy(x: placement.origin.x, y: placement.origin.y)
        context.scaleBy(x: placement.scale, y: placement.scale)
        context.setShouldAntialias(false)

        for cell in frame.cells where cell.b != frame.background {
            context.setFillColor(color(cell.b).cgColor)
            context.fill(rect(for: cell, columns: frame.width))
        }
        for cell in frame.cells {
            guard let symbol = cell.t else { continue }
            let rect = rect(for: cell, columns: frame.width)
            let foreground = color(cell.f)
            context.setFillColor(foreground.cgColor)
            switch symbol {
            case "▌":
                context.fill(CGRect(x: rect.minX, y: rect.minY, width: rect.width / 2, height: rect.height))
            case "▀", "▄":
                context.fill(CGRect(x: rect.minX, y: symbol == "▀" ? rect.minY : rect.midY,
                                    width: rect.width, height: rect.height / 2))
            case "·":
                context.fill(CGRect(x: rect.midX - 1, y: rect.midY - 1, width: 2, height: 2))
            case "─", "│", "┌", "┐", "└", "┘":
                drawBorder(symbol, in: rect, context: context)
            default:
                let text = symbol as NSString
                let attributes: [NSAttributedString.Key: Any] = [.font: cell.w > 1 ? wideFont : font,
                                                                .foregroundColor: foreground]
                let size = text.size(withAttributes: attributes)
                context.setShouldAntialias(true)
                text.draw(at: CGPoint(x: rect.midX - size.width / 2, y: rect.midY - size.height / 2),
                          withAttributes: attributes)
                context.setShouldAntialias(false)
            }
        }
    }

    private func rect(for cell: GameCell, columns: Int) -> CGRect {
        CGRect(x: CGFloat(cell.i % columns) * cellWidth, y: CGFloat(cell.i / columns) * cellHeight,
               width: CGFloat(cell.w) * cellWidth, height: cellHeight)
    }

    private func drawBorder(_ symbol: String, in rect: CGRect, context: CGContext) {
        let left = ["─", "┐", "┘"].contains(symbol) ? rect.minX : rect.midX
        let right = ["─", "┌", "└"].contains(symbol) ? rect.maxX : rect.midX
        let top = ["│", "└", "┘"].contains(symbol) ? rect.minY : rect.midY
        let bottom = ["│", "┌", "┐"].contains(symbol) ? rect.maxY : rect.midY
        let thickness: CGFloat = 1
        if left < right {
            context.fill(CGRect(x: left - thickness / 2, y: rect.midY - thickness / 2,
                                width: right - left + thickness, height: thickness))
        }
        if top < bottom {
            context.fill(CGRect(x: rect.midX - thickness / 2, y: top - thickness / 2,
                                width: thickness, height: bottom - top + thickness))
        }
    }

    override func mouseDown(with event: NSEvent) {
        window?.makeFirstResponder(self)
        guard let frame = gameFrame else { return }
        let placement = layout(for: frame)
        let point = convert(event.locationInWindow, from: nil)
        let column = floor((point.x - placement.origin.x) / (cellWidth * placement.scale))
        let row = floor((point.y - placement.origin.y) / (cellHeight * placement.scale))
        if column >= 0, row >= 0, column < CGFloat(frame.width), row < CGFloat(frame.height) {
            send("click \(Int(column)) \(Int(row))")
        }
    }

    override func keyDown(with event: NSEvent) {
        switch event.keyCode {
        case 36, 76: send("enter")
        case 53: send("escape")
        case 123: send("left")
        case 124: send("right")
        case 125: send("down")
        case 126: send("up")
        default:
            guard let key = event.charactersIgnoringModifiers?.lowercased(),
                  key.count == 1, "adwsprmq ".contains(key) else { return }
            send(key)
        }
    }
}

final class GameApp: NSObject, NSApplicationDelegate {
    private let game = Process()
    private let input = Pipe()
    private let output = Pipe()
    private let view = GameView()
    private let window = NSWindow(contentRect: NSRect(x: 0, y: 0, width: 728, height: 728),
                                  styleMask: [.titled, .closable, .miniaturizable, .resizable],
                                  backing: .buffered, defer: false)
    private let decoder = JSONDecoder()
    private var incoming = Data()

    func applicationDidFinishLaunching(_ notification: Notification) {
        configureMenu()
        window.title = "픽셀 테트리스"
        window.appearance = NSAppearance(named: .darkAqua)
        window.contentMinSize = NSSize(width: 460, height: 460)
        window.contentView = view
        window.center()
        window.makeKeyAndOrderFront(nil)
        window.makeFirstResponder(view)
        NSApp.activate()

        guard let engine = Bundle.main.url(forResource: "pixel-tetris", withExtension: nil) else {
            showLaunchError()
            return
        }
        view.send = { [weak self] message in
            guard let self, game.isRunning else { return }
            try? input.fileHandleForWriting.write(contentsOf: Data((message + "\n").utf8))
        }

        output.fileHandleForReading.readabilityHandler = { [weak self] handle in
            let chunk = handle.availableData
            if chunk.isEmpty {
                handle.readabilityHandler = nil
                return
            }
            DispatchQueue.main.async { self?.receive(chunk) }
        }
        game.executableURL = engine
        game.arguments = ["--window"]
        game.standardInput = input
        game.standardOutput = output
        game.terminationHandler = { [weak self] process in
            DispatchQueue.main.async {
                if process.terminationStatus != 0 { self?.showLaunchError() }
                else { NSApp.terminate(nil) }
            }
        }
        do { try game.run() }
        catch { showLaunchError() }
    }

    private func configureMenu() {
        let menu = NSMenu()
        let app = NSMenuItem()
        let appMenu = NSMenu()
        appMenu.addItem(withTitle: "픽셀 테트리스 종료", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
        app.submenu = appMenu
        menu.addItem(app)
        NSApp.mainMenu = menu
    }

    private func receive(_ chunk: Data) {
        incoming.append(chunk)
        while let newline = incoming.firstIndex(of: 10) {
            let line = incoming[..<newline]
            if let frame = try? decoder.decode(GameFrame.self, from: line) {
                view.gameFrame = frame
            }
            incoming.removeSubrange(...newline)
        }
    }

    private func showLaunchError() {
        let alert = NSAlert()
        alert.messageText = "픽셀 테트리스를 열 수 없습니다"
        alert.informativeText = "앱을 닫고 다시 실행해 주세요."
        alert.addButton(withTitle: "닫기")
        alert.runModal()
        NSApp.terminate(nil)
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool { true }

    func applicationWillTerminate(_ notification: Notification) {
        output.fileHandleForReading.readabilityHandler = nil
        try? input.fileHandleForWriting.close()
    }
}

let app = NSApplication.shared
let delegate = GameApp()
app.delegate = delegate
app.setActivationPolicy(.regular)
app.run()
