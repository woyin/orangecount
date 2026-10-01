// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

import Foundation

@main
struct ClientBurstBenchmark {
    @MainActor static func main() async throws {
        let client = NativeLedgerClient()
        let opened = try await client.open(path: CommandLine.arguments[1])
        var superseded: [Task<LedgerView, Error>] = []
        for _ in 0..<20 {
            superseded.append(Task { try await client.read(handle: opened.handle, request: ViewRequest(view: "dashboard")) })
            await Task.yield()
        }
        let started = Date()
        for task in superseded { task.cancel() }
        let latest = try await client.read(handle: opened.handle, request: ViewRequest(view: "dashboard", currency: "USD"))
        let elapsed = Date().timeIntervalSince(started) * 1000
        var cancelled = 0
        for task in superseded {
            do { _ = try await task.value }
            catch is CancellationError { cancelled += 1 }
        }
        await client.close(handle: opened.handle)
        guard latest.state.valid else { throw BridgeError.remote("benchmark", "Invalid synthetic input") }
        print("superseded=20 cancelled_before_execution=\(cancelled) latest_response_ms=\(Int(elapsed))")
    }
}
