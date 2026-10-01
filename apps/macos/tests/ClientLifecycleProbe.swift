// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

import Darwin
import Foundation

@main
struct ClientLifecycleProbe {
    static func residentBytes() throws -> UInt64 {
        var info = mach_task_basic_info_data_t()
        var count = mach_msg_type_number_t(MemoryLayout<mach_task_basic_info_data_t>.size / MemoryLayout<integer_t>.size)
        let status = withUnsafeMutablePointer(to: &info) { pointer in
            pointer.withMemoryRebound(to: integer_t.self, capacity: Int(count)) { values in
                task_info(mach_task_self_, task_flavor_t(MACH_TASK_BASIC_INFO), values, &count)
            }
        }
        guard status == KERN_SUCCESS else { throw BridgeError.remote("probe", "Cannot read process memory") }
        return UInt64(info.resident_size)
    }

    static func main() async throws {
        let path = CommandLine.arguments[1]
        let cycles = CommandLine.arguments.count > 2 ? Int(CommandLine.arguments[2]) ?? 20 : 20
        guard cycles >= 1 && cycles <= 1000 else { throw BridgeError.remote("probe", "Cycles must be 1...1000") }
        let client = NativeLedgerClient()
        var samples: [UInt64] = []
        let started = Date()
        for cycle in 1...cycles {
            let opened = try await client.open(path: path)
            do {
                for view in ["balance_sheet", "income_statement", "journal", "dashboard"] {
                    let response = try await client.read(handle: opened.handle, request: ViewRequest(view: view, currency: view == "dashboard" || view == "journal" ? "" : "native"))
                    guard response.state.valid && response.state.build_count == 1 else { throw BridgeError.remote("probe", "Expected one valid build per session") }
                }
                await client.close(handle: opened.handle)
                do {
                    _ = try await client.poll(handle: opened.handle, after: 0)
                    throw BridgeError.remote("probe", "Closed handle still responds")
                } catch BridgeError.remote("session-closed", _) {}
            } catch {
                await client.close(handle: opened.handle)
                throw error
            }
            samples.append(try residentBytes())
            if cycle == 1 || cycle % 10 == 0 { print("cycle=\(cycle) resident_bytes=\(samples.last!)") }
        }
        print("cycles=\(cycles) elapsed_seconds=\(Int(Date().timeIntervalSince(started))) rss_samples_bytes=\(samples)")
    }
}
