import AVFoundation
import Combine
import SwiftUI
import UIKit

private let backdrop = Color(rgb: 0x101222)
private let well = Color(rgb: 0x090b16)
private let ink = Color(rgb: 0xe4dbfb)
private let accent = Color(rgb: 0xdd8dff)
private let frameColor = Color(rgb: 0x786799)
private let gold = Color(rgb: 0xffd553)

private let fills = [0x26c6da, 0x6486ff, 0xffa44b, 0xffd553, 0x63d98b, 0xcc7cf4, 0xff6786]
private let edges = [0xa0f2ff, 0xb4c6ff, 0xffd39b, 0xffefae, 0xb8f9bf, 0xedd0ff, 0xffbdc9]
private let ghostFills = [0x102f3b, 0x1a243f, 0x382a20, 0x383321, 0x193227, 0x2c223e, 0x381e2b]
private let ghostEdges = [0x25606c, 0x3a4e80, 0x795437, 0x776637, 0x376b50, 0x694778, 0x783f51]

private extension Color {
    init(rgb: Int) {
        self.init(red: Double((rgb >> 16) & 255) / 255,
                  green: Double((rgb >> 8) & 255) / 255,
                  blue: Double(rgb & 255) / 255)
    }
}

private struct Frame: Decodable {
    let cells, next: [Int]
    let score, lines, level, combo, comboPoints, multi, linePoints: Int
    let paused, over, bright: Bool
    let sounds: [String]?
    let drop: DropFrame?
}

private struct DropFrame: Decodable {
    let id: Int
    let cells: [[Int]]
    let offset: CGFloat
}

@main
struct PixelTetrisApp: App {
    var body: some Scene {
        WindowGroup { GameView().preferredColorScheme(.dark) }
    }
}

private struct GameView: View {
    @Environment(\.scenePhase) private var scenePhase
    @State private var frame: Frame?
    @State private var swipe = BoardSwipe()
    @GestureState private var touching = false
    @State private var rotationFeedback = UIImpactFeedbackGenerator(style: .light)
    @State private var landingFeedback = UIImpactFeedbackGenerator(style: .medium)
    @State private var clearFeedback = UINotificationFeedbackGenerator()
    @State private var players: [String: AVAudioPlayer] = [:]
    @AppStorage("soundEnabled") private var soundEnabled = true
    private let clock = Timer.publish(every: 1.0 / 60.0, on: .main, in: .common).autoconnect()

    var body: some View {
        GeometryReader { geometry in
            let panelWidth = min(500, geometry.size.width - 24)
            let cell = floor(min(panelWidth / 10, (geometry.size.height - 157) / 20))
            VStack(spacing: 8) {
                header.frame(height: 44)
                statusBar.frame(height: 49)
                board(cell)
                footer.frame(height: 32)
            }
            .frame(width: panelWidth)
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
        .background(backdrop.ignoresSafeArea())
        .onReceive(clock) { _ in
            if scenePhase == .active && frame != nil { action("tick") }
        }
        .onChange(of: scenePhase) { _, phase in
            if phase != .active {
                swipe = BoardSwipe()
                action("background")
                for player in players.values { player.stop() }
            }
        }
    }

    private var header: some View {
        HStack(spacing: 8) {
            Text("▰  픽셀 테트리스")
                .font(.system(size: 19, weight: .heavy, design: .monospaced))
                .foregroundStyle(accent)
                .lineLimit(1)
                .minimumScaleFactor(0.65)
            Spacer(minLength: 0)
            if let frame {
                iconButton(frame.paused ? "play.fill" : "pause.fill", frame.paused ? "계속" : "일시정지") {
                    action("pause")
                }
                .disabled(frame.over)
                iconButton("arrow.counterclockwise", "다시 시작") { action("restart") }
            }
            iconButton(soundEnabled ? "speaker.wave.2.fill" : "speaker.slash.fill",
                       soundEnabled ? "소리 끄기" : "소리 켜기") {
                soundEnabled.toggle()
                if !soundEnabled { for player in players.values { player.stop() } }
            }
        }
        .frame(maxWidth: 500)
    }

    private func board(_ cell: CGFloat) -> some View {
        Canvas { context, _ in
            for row in 0..<20 {
                for col in 0..<10 {
                    let rect = CGRect(x: CGFloat(col) * cell, y: CGFloat(row) * cell, width: cell, height: cell)
                    let id = frame?.cells[row * 10 + col] ?? 0
                    if id == 0 {
                        let dot = CGRect(x: rect.midX - 1, y: rect.midY - 1, width: 2, height: 2)
                        context.fill(Path(dot), with: .color(Color(rgb: 0x252840)))
                    } else {
                        tile(context, rect, id)
                    }
                }
            }
            if let drop = frame?.drop {
                if drop.offset > 1 {
                    for point in drop.cells {
                        tile(context, fallingTile(point, cell, offset: drop.offset - 1.25), -drop.id)
                    }
                }
                for point in drop.cells {
                    tile(context, fallingTile(point, cell, offset: drop.offset), drop.id)
                }
            }
        }
        .frame(width: 10 * cell, height: 20 * cell)
        .clipped()
        .background(well)
        .overlay { Rectangle().strokeBorder(frameColor, lineWidth: 1) }
        .contentShape(Rectangle())
        .gesture(boardGesture(cell), including: frame != nil && frame?.paused == false && frame?.over == false ? .gesture : .none)
        .onChange(of: touching) { _, active in
            if !active { swipe = BoardSwipe() }
        }
        .overlay {
            if frame == nil {
                overlay("픽셀 테트리스", "게임 시작", "start", icon: "play.fill")
            } else if frame?.over == true {
                overlay("게임 종료", "다시 시작", "restart", icon: "arrow.counterclockwise")
            } else if frame?.paused == true {
                overlay("일시정지", "계속하기", "pause", icon: "play.fill")
            }
        }
    }

    private func fallingTile(_ point: [Int], _ cell: CGFloat, offset: CGFloat) -> CGRect {
        CGRect(x: CGFloat(point[0]) * cell, y: (CGFloat(point[1]) + offset) * cell,
               width: cell, height: cell)
    }

    private func overlay(_ title: String, _ button: String, _ command: String, icon: String) -> some View {
        VStack(spacing: 16) {
            Text(title)
                .font(.system(size: 19, weight: .heavy, design: .monospaced))
                .foregroundStyle(ink)
                .lineLimit(1)
                .minimumScaleFactor(0.7)
            Button { action(command) } label: {
                Label(button, systemImage: icon)
                    .font(.headline)
                    .foregroundStyle(backdrop)
                    .padding(.horizontal, 18)
                    .frame(minHeight: 48)
                    .background(accent)
            }
            .buttonStyle(PixelButton())
            .accessibilityIdentifier(command)
        }
        .padding(20)
        .background(backdrop)
        .overlay { Rectangle().strokeBorder(frameColor, lineWidth: 1) }
        .padding(5)
    }

    private var statusBar: some View {
        HStack(spacing: 12) {
            stat("점수", frame?.score ?? 0)
                .frame(maxWidth: .infinity)
            stat("줄", frame?.lines ?? 0)
                .frame(maxWidth: .infinity)
            stat("레벨", frame?.level ?? 1)
                .frame(maxWidth: .infinity)
            Rectangle()
                .fill(frameColor)
                .frame(width: 1, height: 28)
                .accessibilityHidden(true)
            VStack(spacing: 2) {
                Text("다음 블록").font(.caption2)
                Canvas { context, size in
                    guard let next = frame?.next else { return }
                    let coordinates = stride(from: 1, to: next.count, by: 2).map { (next[$0], next[$0 + 1]) }
                    guard let minX = coordinates.map(\.0).min(), let maxX = coordinates.map(\.0).max(),
                          let minY = coordinates.map(\.1).min(), let maxY = coordinates.map(\.1).max() else { return }
                    let cell: CGFloat = 10
                    let x = (size.width - CGFloat(maxX - minX + 1) * cell) / 2
                    let y = (size.height - CGFloat(maxY - minY + 1) * cell) / 2
                    for (col, row) in coordinates {
                        tile(context, CGRect(x: x + CGFloat(col - minX) * cell,
                                             y: y + CGFloat(row - minY) * cell, width: cell, height: cell), next[0])
                    }
                }
                .frame(width: 62, height: 22)
            }
        }
        .padding(.horizontal, 8)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .foregroundStyle(ink)
        .background(well)
        .overlay { Rectangle().strokeBorder(frameColor, lineWidth: 1) }
    }

    private func stat(_ label: String, _ value: Int) -> some View {
        VStack(spacing: 3) {
            Text(label).font(.caption2)
            Text(value, format: .number.grouping(.never))
                .font(.system(size: 16, weight: .bold, design: .monospaced))
                .foregroundStyle(accent)
                .lineLimit(1)
                .minimumScaleFactor(0.5)
        }
    }

    private var footer: some View {
        Group {
            if let frame, frame.comboPoints > 0 || frame.multi > 1 {
                HStack(spacing: 8) {
                    if frame.comboPoints > 0 {
                        reward("\(frame.combo) 콤보", frame.comboPoints, accent, bright: frame.bright)
                    }
                    if frame.multi > 1 {
                        reward("\(frame.multi)줄 삭제", frame.linePoints, gold, bright: frame.bright)
                    }
                }
            } else {
                Text("탭 회전  ·  좌우로 이동\n아래로 끌어 내리기  ·  빠르게 쓸어 즉시 낙하")
                    .font(.system(size: 12, weight: .medium))
                    .foregroundStyle(ink.opacity(0.72))
                    .multilineTextAlignment(.center)
            }
        }
    }

    private func reward(_ label: String, _ points: Int, _ tint: Color, bright: Bool) -> some View {
        HStack(spacing: 5) {
            Text(label)
            Text("+\(points)").monospacedDigit()
        }
        .font(.system(size: 12, weight: .bold))
        .foregroundStyle(bright ? backdrop : tint)
        .padding(6)
        .frame(maxWidth: .infinity)
        .background(bright ? tint : well)
        .overlay { Rectangle().strokeBorder(tint, lineWidth: 1) }
    }

    private func boardGesture(_ cell: CGFloat) -> some Gesture {
        DragGesture(minimumDistance: 0)
            .updating($touching) { _, state, _ in state = true }
            .onChanged { value in
                let commands = swipe.update(translation: value.translation, predicted: value.predictedEndTranslation,
                                            time: value.time, cell: cell)
                for command in commands { action(command) }
            }
            .onEnded { value in
                let commands = swipe.finish(translation: value.translation, predicted: value.predictedEndTranslation,
                                            time: value.time, cell: cell)
                for command in commands { action(command) }
            }
    }

    private func iconButton(_ icon: String, _ label: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Image(systemName: icon)
                .font(.system(size: 15, weight: .semibold))
                .foregroundStyle(accent)
                .frame(width: 38, height: 44)
                .background(well)
        }
        .buttonStyle(PixelButton())
        .accessibilityLabel(label)
    }

    private func tile(_ context: GraphicsContext, _ rect: CGRect, _ id: Int) {
        let fill, edge: Color
        switch id {
        case 8: (fill, edge) = (ink, accent)
        case 9: (fill, edge) = (frameColor, ink)
        case -7 ... -1: (fill, edge) = (Color(rgb: ghostFills[-id - 1]), Color(rgb: ghostEdges[-id - 1]))
        case 1 ... 7: (fill, edge) = (Color(rgb: fills[id - 1]), Color(rgb: edges[id - 1]))
        default: return
        }
        context.fill(Path(rect), with: .color(fill))
        let highlight = CGRect(x: rect.minX, y: rect.minY, width: max(2, floor(rect.width / 4)), height: rect.height)
        context.fill(Path(highlight), with: .color(edge))
    }

    private func action(_ command: String) {
        guard let result = command.withCString({ PTGame($0) }) else { return }
        defer { free(result) }
        let data = Data(bytes: result, count: strlen(result))
        guard let updated = try? JSONDecoder().decode(Frame.self, from: data) else { return }
        frame = updated
        let effects = updated.sounds ?? []
        if command == "rotate" {
            rotationFeedback.impactOccurred()
        }
        if effects.contains("clear") || effects.contains("multi") {
            clearFeedback.notificationOccurred(.success)
        } else if effects.contains("lock") || effects.contains("over") {
            landingFeedback.impactOccurred()
        }
        guard soundEnabled else { return }
        for effect in effects {
            if players[effect] == nil, let url = Bundle.main.url(forResource: effect, withExtension: "wav") {
                players[effect] = try? AVAudioPlayer(contentsOf: url)
            }
            let player = players[effect]
            player?.currentTime = 0
            player?.play()
        }
    }
}

private struct PixelButton: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label.contentShape(Rectangle()).opacity(configuration.isPressed ? 0.7 : 1)
    }
}
