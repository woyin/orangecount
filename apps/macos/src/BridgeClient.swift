import Foundation
import OrangeCountBridge

enum BridgeError: Error, LocalizedError {
    case nullPointerFromGo
    case jsonDecodeError(String)
    case protocolMismatch(Int)
    case remote(String, String)

    var errorDescription: String? {
        switch self {
        case .nullPointerFromGo:
            return "Go runtime returned a null pointer"
        case .jsonDecodeError(let detail):
            return "Failed to decode JSON from Go core: \(detail)"
        case .protocolMismatch(let version):
            return "Unsupported ledger protocol version: \(version)"
        case .remote(_, let message):
            return message
        }
    }
    func message(locale: String) -> String {
        guard locale == "zh-CN" else { return errorDescription ?? "Unable to read ledger" }
        switch self {
        case .nullPointerFromGo: return "账本引擎未返回响应，请重新打开账本。"
        case .jsonDecodeError: return "账本响应格式不正确，请重新加载或更新应用。"
        case .protocolMismatch(let version): return "账本协议版本不兼容（\(version)），请使用同一构建版本的应用。"
        case .remote(let code, let message):
            switch code {
            case "invalid-request": return "请求无效，请检查日期范围等筛选条件。"
            case "session-closed": return "账本会话已关闭，请重新打开账本。"
            case "snapshot-changed": return "账本已更新，请刷新当前页面。"
            case "no-snapshot": return "尚无有效账本快照，请先修复账本错误。"
            case "text-budget-exceeded": return "查询文本超过64 KiB，请缩短查询。"
            case "account-not-found": return "当前快照中没有此账户，可返回并重新选择。"
            case "commodity-not-found": return "当前快照中没有此商品，可返回价格目录并重新选择。"
            case "result-budget-exceeded": return "结果超出大小限制，请减少每页数量或缩小账户、日期范围。"
            case "ambiguous-directory": return "目录中没有唯一账本入口，请选择具体文件。"
            case "encoding", "internal": return "账本引擎无法完成请求，请重新加载。"
            default: return message
            }
        }
    }

}

// Memory-safe C-ABI wrapper implementing the RAII protocol specified in native-desktop-app-plan.md
final class BridgeClient: @unchecked Sendable {
    static let shared = BridgeClient()
    private init() {}

    func openLedger(path: String, semantics: String = "legacy", inputIdentity: String = "") throws -> OpenLedgerResult {
        let data = try JSONEncoder().encode(["semantics_profile": semantics, "input_identity": inputIdentity])
        let options = String(decoding: data, as: UTF8.self)
        return try path.withCString { ptr in
            try options.withCString { optionsPtr in
                try sessionResponse { OC_OpenLedgerWithOptions(UnsafeMutablePointer(mutating: ptr), UnsafeMutablePointer(mutating: optionsPtr)) }
            }
        }
    }

    func pollState(handle: UInt64, after: UInt64) throws -> PollResult {
        try sessionResponse { OC_PollState(handle, after) }
    }

    func readView(handle: UInt64, request: ViewRequest) throws -> LedgerView {
        let data = try JSONEncoder().encode(request)
        let json = String(decoding: data, as: UTF8.self)
        let response: LedgerView = try json.withCString { ptr in
            try sessionResponse { OC_ReadView(handle, UnsafeMutablePointer(mutating: ptr)) }
        }
        if request.view == "query", response.state.has_snapshot {
            guard response.query_catalogue?.snapshot_id == response.state.snapshot_id else {
                throw BridgeError.jsonDecodeError("Saved query catalogue does not match view snapshot")
            }
        }
        if request.document_id == nil, request.view == "journal", ["all", "config"].contains(request.journal_mode ?? ""), response.state.has_snapshot {
            guard response.directives?.snapshot_id == response.state.snapshot_id,
                  response.directives?.mode == request.journal_mode, response.journal == nil else {
                throw BridgeError.jsonDecodeError("Directive page does not match captured view")
            }
        }
        if let id = request.document_id {
            guard response.document_action?.id == id, response.document_action?.path.hasPrefix("/") == true, response.state.snapshot_id == request.snapshot_id else { throw BridgeError.jsonDecodeError("Attachment authorization does not match request") }
        }
        return response
    }

    func reloadLedger(handle: UInt64) throws -> LedgerState {
        try sessionResponse { OC_ReloadLedger(handle) }
    }

    func executeQuery(handle: UInt64, request: QueryRequest) throws -> QueryResponse {
        guard request.text.utf8.count <= 64 * 1024 else { throw BridgeError.remote("text-budget-exceeded", "Query text exceeds 64 KiB") }
        let data = try JSONEncoder().encode(request)
        let json = String(decoding: data, as: UTF8.self)
        let response: QueryResponse = try json.withCString { ptr in
            try sessionResponse { OC_ExecuteQuery(handle, UnsafeMutablePointer(mutating: ptr)) }
        }
        guard response.context.request_id == request.request_id, response.context.snapshot_id == request.snapshot_id,
              response.context.text == request.text else { throw BridgeError.jsonDecodeError("Query response does not match its captured request") }
        return response
    }

    func cancelQuery(handle: UInt64, requestID: UInt64) throws {
        struct Accepted: Decodable { let accepted: Bool }
        let result: Accepted = try sessionResponse { OC_CancelQuery(handle, requestID) }
        guard result.accepted else { throw BridgeError.jsonDecodeError("Query cancellation was not accepted") }
    }

    func captureQueryExport(handle: UInt64, request: QueryExportRequest) throws -> QueryExportCapture {
        let data = try JSONEncoder().encode(request)
        let json = String(decoding: data, as: UTF8.self)
        let response: QueryExportCapture = try json.withCString { ptr in
            try sessionResponse { OC_CaptureQueryExport(handle, UnsafeMutablePointer(mutating: ptr)) }
        }
        guard response.context.snapshot_id == request.snapshot_id,
              response.context.request_id == request.request_id,
              response.context.fingerprint == request.fingerprint,
              response.format == request.format,
              !response.protected_paths.isEmpty,
              Set(response.protected_paths).count == response.protected_paths.count,
              response.protected_paths.allSatisfy({ $0.hasPrefix("/") }),
              response.content.count <= 16 * 1024 * 1024 else {
            throw BridgeError.jsonDecodeError("Query export does not match captured execution or exceeds the byte budget")
        }
        return response
    }

    func exportProtection(handle: UInt64) throws -> ExportProtection {
        let response: ExportProtection = try sessionResponse { OC_ExportProtection(handle) }
        guard !response.protected_paths.isEmpty,
              Set(response.protected_paths).count == response.protected_paths.count,
              response.protected_paths.allSatisfy({ $0.hasPrefix("/") }) else {
            throw BridgeError.jsonDecodeError("Invalid export protection paths")
        }
        return response
    }

    func closeLedger(handle: UInt64) throws {
        struct Closed: Decodable { let closed: Bool }
        let _: Closed = try sessionResponse { OC_CloseLedger(handle) }
    }

    private struct ResponseError: Decodable {
        let code: String
        let message: String
    }

    private struct Response<Value: Decodable>: Decodable {
        let protocol_version: Int
        let data: Value?
        let error: ResponseError?
        enum CodingKeys: CodingKey { case protocol_version, data, error }
        init(from decoder: Decoder) throws {
            let values = try decoder.container(keyedBy: CodingKeys.self)
            protocol_version = try values.decode(Int.self, forKey: .protocol_version)
            guard protocol_version == 2 else { throw BridgeError.protocolMismatch(protocol_version) }
            error = try values.decodeIfPresent(ResponseError.self, forKey: .error)
            data = try values.decodeIfPresent(Value.self, forKey: .data)
        }
    }

    // Shared by live C calls and protocol/decoder probes. It does not log data.
    static func decodeResponse<Value: Decodable>(_ data: Data, as type: Value.Type) throws -> Value {
        do {
            let response = try JSONDecoder().decode(Response<Value>.self, from: data)
            if let error = response.error { throw BridgeError.remote(error.code, error.message) }
            guard let value = response.data else { throw BridgeError.jsonDecodeError("Missing response data") }
            return value
        } catch {
            if let error = error as? BridgeError { throw error }
            throw BridgeError.jsonDecodeError(error.localizedDescription)
        }
    }

    private func sessionResponse<Value: Decodable>(_ call: () -> UnsafeMutablePointer<CChar>?) throws -> Value {
        try withGoResponse(call) { try Self.decodeResponse($0, as: Value.self) }
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
            if let error = error as? BridgeError { throw error }
            // Raw responses contain private ledger data. Report only the decode category.
            throw BridgeError.jsonDecodeError(error.localizedDescription)
        }
    }
}
