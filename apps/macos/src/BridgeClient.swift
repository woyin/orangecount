import Foundation
import OrangeCountBridge

enum BridgeError: Error, LocalizedError {
    case nullPointerFromGo
    case jsonDecodeError(String)

    var errorDescription: String? {
        switch self {
        case .nullPointerFromGo:
            return "Go runtime returned a null pointer"
        case .jsonDecodeError(let detail):
            return "Failed to decode JSON from Go core: \(detail)"
        }
    }
}

// Memory-safe C-ABI wrapper implementing the RAII protocol specified in native-desktop-app-plan.md
final class BridgeClient {
    static let shared = BridgeClient()
    private init() {}

    func ping() -> String {
        guard let ptr = OC_Ping() else { return "Ping failed: null pointer" }
        defer { OC_FreeString(ptr) }
        return String(cString: ptr)
    }

    func loadSnapshot(path: String) throws -> LedgerSummary {
        try path.withCString { cPath in
            try withGoResponse {
                OC_LoadSnapshot(UnsafeMutablePointer(mutating: cPath))
            } decode: { data in
                try JSONDecoder().decode(LedgerSummary.self, from: data)
            }
        }
    }

    func getTreeReport(path: String, reportType: String) throws -> TreeReportPayload {
        try path.withCString { cPath in
            try reportType.withCString { cReport in
                try withGoResponse {
                    OC_GetTreeReport(UnsafeMutablePointer(mutating: cPath), UnsafeMutablePointer(mutating: cReport))
                } decode: { data in
                    try JSONDecoder().decode(TreeReportPayload.self, from: data)
                }
            }
        }
    }

    func getJournal(path: String, filterJSON: String = "") throws -> JournalPayload {
        try path.withCString { cPath in
            try filterJSON.withCString { cFilter in
                try withGoResponse {
                    OC_GetJournal(UnsafeMutablePointer(mutating: cPath), UnsafeMutablePointer(mutating: cFilter))
                } decode: { data in
                    try JSONDecoder().decode(JournalPayload.self, from: data)
                }
            }
        }
    }

    func getHistoricalTrends(path: String) throws -> HistoricalTrendsPayload {
        try path.withCString { cPath in
            try withGoResponse {
                OC_GetHistoricalTrends(UnsafeMutablePointer(mutating: cPath))
            } decode: { data in
                try JSONDecoder().decode(HistoricalTrendsPayload.self, from: data)
            }
        }
    }

    func getLedgerRevision(path: String) -> String {
        path.withCString { cPath in
            guard let ptr = OC_GetLedgerRevision(UnsafeMutablePointer(mutating: cPath)) else {
                return "rev:0"
            }
            defer { OC_FreeString(ptr) }
            return String(cString: ptr)
        }
    }

    private func withGoResponse<T>(_ call: () -> UnsafeMutablePointer<CChar>?, decode: (Data) throws -> T) throws -> T {
        guard let cPtr = call() else {
            throw BridgeError.nullPointerFromGo
        }
        defer { OC_FreeString(cPtr) }
        let data = Data(bytes: cPtr, count: strlen(cPtr))
        do {
            return try decode(data)
        } catch {
            let rawStr = String(data: data, encoding: .utf8) ?? "<invalid utf8>"
            throw BridgeError.jsonDecodeError("\(error.localizedDescription)\nRaw JSON: \(rawStr)")
        }
    }
}
