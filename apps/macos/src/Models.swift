import Foundation

// Summary payload returned from Go Core
struct LedgerSummary: Codable {
    var valid: Bool
    var entry_path: String?
    var title: String?
    var operating_currencies: [String]?
    var accounts: [String]
    var entries_count: Int
    var error_count: Int
    var errors: [String]
}

// Tree report response
struct TreeReportPayload: Codable {
    var report_type: String
    var operating_currencies: [String]?
    var root_nodes: [AccountTreeNode]
    var error: String?
}

// Hierarchical account node directly matching internal/report.AccountTreeNode
final class AccountTreeNode: Codable, Identifiable, ObservableObject {
    var id: String { full_name }
    let name: String
    let full_name: String
    let depth: int_fast32_t
    let is_leaf: Bool
    let explicit: Bool
    let direct_balances: [String: String]
    let subtree_balances: [String: String]
    let children: [AccountTreeNode]

    init(name: String, fullName: String, depth: int_fast32_t, isLeaf: Bool, explicit: Bool, directBalances: [String: String], subtreeBalances: [String: String], children: [AccountTreeNode]) {
        self.name = name
        self.full_name = fullName
        self.depth = depth
        self.is_leaf = isLeaf
        self.explicit = explicit
        self.direct_balances = directBalances
        self.subtree_balances = subtreeBalances
        self.children = children
    }
}

// Journal models
struct JournalPosting: Codable, Identifiable {
    var id: String { "\(account)_\(units_number)_\(units_currency)" }
    let account: String
    let units_number: String
    let units_currency: String
    let cost: String?
    let price: String?
}

struct JournalTransaction: Codable, Identifiable {
    let id: String
    let date: String
    let flag: String
    let payee: String
    let narration: String
    let tags: [String]
    let links: [String]
    let postings: [JournalPosting]
}

struct JournalPayload: Codable {
    let total_count: Int
    let transactions: [JournalTransaction]
    let error: String?
}

// Chart models for Swift Charts
struct TrendPoint: Codable, Identifiable {
    var id: String { date }
    let date: String
    let value: Double
    let currency: String
}

struct CashFlowBar: Codable, Identifiable {
    var id: String { month }
    let month: String
    let income: Double
    let expenses: Double
    let net: Double
    let currency: String
}

struct HistoricalTrendsPayload: Codable {
    let operating_currency: String
    let net_worth_points: [TrendPoint]
    let monthly_cash_flows: [CashFlowBar]
    let error: String?
}
