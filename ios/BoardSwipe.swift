import Foundation

struct BoardSwipe {
    private enum Axis { case horizontal, vertical }
    private static let tapSlop: CGFloat = 9
    private var axis: Axis?
    private var started: Date?
    private var anchor: CGFloat = 0
    private var moved = false

    mutating func update(translation: CGSize, predicted: CGSize, time: Date, cell: CGFloat) -> [String] {
        if started == nil { started = time }
        chooseAxis(translation)
        switch axis {
        case .horizontal:
            return steps(toward: translation.width, size: cell * 0.8, positive: "right", negative: "left")
        case .vertical:
            guard let started, time.timeIntervalSince(started) >= 0.14,
                  !isFlick(translation, predicted: predicted, cell: cell) else { return [] }
            return steps(toward: max(anchor, translation.height), size: cell, positive: "down", negative: "down")
        case nil:
            return []
        }
    }

    mutating func finish(translation: CGSize, predicted: CGSize, time: Date, cell: CGFloat) -> [String] {
        defer { self = Self() }
        chooseAxis(translation)
        switch axis {
        case .horizontal:
            let moves = steps(toward: translation.width, size: cell * 0.8, positive: "right", negative: "left")
            if moved { return moves }
            return [translation.width < 0 ? "left" : "right"]
        case .vertical:
            let elapsed = time.timeIntervalSince(started ?? time)
            let quickSwipe = elapsed <= 0.18 && translation.height >= max(48, cell * 1.8)
            if isFlick(translation, predicted: predicted, cell: cell) || quickSwipe {
                return ["drop"]
            }
            return steps(toward: max(anchor, translation.height), size: cell, positive: "down", negative: "down")
        case nil:
            if time.timeIntervalSince(started ?? time) <= 0.28 { return ["rotate"] }
            return []
        }
    }

    private mutating func chooseAxis(_ translation: CGSize) {
        guard axis == nil else { return }
        let x = abs(translation.width)
        let y = abs(translation.height)
        guard max(x, y) >= Self.tapSlop else { return }
        axis = x >= y ? .horizontal : .vertical
    }

    private func isFlick(_ translation: CGSize, predicted: CGSize, cell: CGFloat) -> Bool {
        translation.height >= max(24, cell * 0.9) &&
            predicted.height - translation.height >= max(36, cell * 1.3)
    }

    private mutating func steps(toward position: CGFloat, size: CGFloat, positive: String, negative: String) -> [String] {
        let count = Int((position - anchor) / size)
        guard count != 0 else { return [] }
        anchor += CGFloat(count) * size
        moved = true
        return Array(repeating: count > 0 ? positive : negative, count: abs(count))
    }
}
