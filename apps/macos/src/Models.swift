import Foundation

struct LedgerState: Codable, Sendable, Equatable {
    let revision: UInt64
    let snapshot_id: String
    let entry_path: String
    let valid: Bool
    let has_snapshot: Bool
    let reloading: Bool
    let build_count: UInt64
    var semantics_profile: String = "legacy"
    var evaluation_revision: String = ""
    var active_extensions: [String] = []
}

struct ViewRequest: Codable, Sendable, Equatable {
    var view: String = "balance_sheet"
    var valuation: String = "at-cost"
    var currency: String = ""
    var locale: String = "zh-CN"
    var snapshot_id: String? = nil
    var date_from: String = ""
    var date_to: String = ""
    var window: String = ""
    var account: String = ""
    var scope: String? = nil
    var group_by: String? = nil
    var holding_section: String? = nil
    var group_id: String? = nil
    var event_id: String? = nil
    var lot_kind: String? = nil
    var price_section: String? = nil
    var price_base: String? = nil
    var price_quote: String? = nil
    var text: String = ""
    var sort: String = ""
    var offset: Int = 0
    var limit: Int = 0
    var journal_mode: String? = nil
    var directive_kind: String? = nil
    var document_id: String? = nil
    var migration_details: Bool = false
}

struct LedgerView: Codable, Sendable {
    let state: LedgerState
    let request: ViewRequest
    let summary: LedgerSummary
    let tree: TreeReportPayload?
    let journal: JournalPayload?
    let trends: HistoricalTrendsPayload?
    var migration: MigrationComparison? = nil
    var account_detail: AccountDetailPayload? = nil
    var holdings: HoldingsPayload? = nil
    var prices: PricesPayload? = nil
    var query_catalogue: QueryCatalogue? = nil
    var directives: DirectivesPayload? = nil
    var document_action: DocumentAction? = nil
}

// A nullable value must have a key on the wire. Synthesis decodes this wrapper
// with decode(_:forKey:), so omitted fields cannot become a valid empty value.
@propertyWrapper
struct RequiredNullable<Value: Codable & Sendable>: Codable, Sendable {
    var wrappedValue: Value?
    init(wrappedValue: Value?) { self.wrappedValue = wrappedValue }
    init(from decoder: Decoder) throws {
        let value = try decoder.singleValueContainer()
        wrappedValue = value.decodeNil() ? nil : try value.decode(Value.self)
    }
    func encode(to encoder: Encoder) throws {
        var value = encoder.singleValueContainer()
        if let wrappedValue { try value.encode(wrappedValue) } else { try value.encodeNil() }
    }
}

enum PriceSection: String, Codable, Sendable { case commodities, history }
enum PriceReason: String, Codable, Sendable { case known = "", unspecifiedPair = "unspecified-pair", missingLocalQuote = "missing-local-quote" }

struct PriceDeclaration: Codable, Sendable {
    let date: String
    let source: SourceLocation
}
struct PriceCommodity: Codable, Sendable, Identifiable {
    let id: String
    let currency: String
    let quote_currencies: [String]
    let accounts: [String]
    let declarations: [PriceDeclaration]
    let price_count: Int
    var validContract: Bool {
        !id.isEmpty && !currency.isEmpty && price_count >= 0 &&
        Set(quote_currencies).count == quote_currencies.count && quote_currencies.allSatisfy { !$0.isEmpty } &&
        Set(accounts).count == accounts.count
    }
}
struct PriceRecord: Codable, Sendable, Identifiable {
    let id: String
    let base: String
    let quote: String
    let date: String
    let rate: DisplayAmount
    @RequiredNullable var value: Double?
    let source_order: Int
    let segment: Int
    let source: SourceLocation
}
struct SelectedPrice: Codable, Sendable {
    let base: String
    let quote: String
    let date: String
    let rate: DisplayAmount
    let inverse: Bool
    let source_base: String
    let source_quote: String
    let age_days: Int
    let source: SourceLocation
}
struct PricesPayload: Codable, Sendable {
    let commodities: [PriceCommodity]
    let commodities_page: ProjectionPage
    @RequiredNullable var commodity: PriceCommodity?
    let base: String
    let quote: String
    let date_from: String
    let date_to: String
    let section: PriceSection
    let records: [PriceRecord]
    let records_page: ProjectionPage
    @RequiredNullable var selected: SelectedPrice?
    let selected_reason: PriceReason
    enum CodingKeys: String, CodingKey {
        case commodities, commodities_page, commodity, base, quote, date_from, date_to, section, records, records_page, selected, selected_reason
    }
    init(from decoder: Decoder) throws {
        let v = try decoder.container(keyedBy: CodingKeys.self)
        commodities = try v.decode([PriceCommodity].self, forKey: .commodities)
        commodities_page = try v.decode(ProjectionPage.self, forKey: .commodities_page)
        _commodity = try v.decode(RequiredNullable<PriceCommodity>.self, forKey: .commodity)
        base = try v.decode(String.self, forKey: .base)
        quote = try v.decode(String.self, forKey: .quote)
        date_from = try v.decode(String.self, forKey: .date_from)
        date_to = try v.decode(String.self, forKey: .date_to)
        section = try v.decode(PriceSection.self, forKey: .section)
        records = try v.decode([PriceRecord].self, forKey: .records)
        records_page = try v.decode(ProjectionPage.self, forKey: .records_page)
        _selected = try v.decode(RequiredNullable<SelectedPrice>.self, forKey: .selected)
        selected_reason = try v.decode(PriceReason.self, forKey: .selected_reason)
        let targetShape = base.isEmpty ? commodity == nil && quote.isEmpty && section == .commodities : commodity?.currency == base && section == .history
        let selectionShape: Bool
        if let chosen = selected {
            selectionShape = selected_reason == .known && !base.isEmpty && !quote.isEmpty && chosen.base == base && chosen.quote == quote &&
                chosen.source_base == (chosen.inverse ? quote : base) && chosen.source_quote == (chosen.inverse ? base : quote) &&
                chosen.age_days >= 0 && !chosen.rate.exact.isEmpty && (date_to.isEmpty || chosen.date <= date_to)
        } else {
            selectionShape = selected_reason == (base.isEmpty || quote.isEmpty ? .unspecifiedPair : .missingLocalQuote)
        }
        guard targetShape, selectionShape, commodity?.validContract ?? true,
              commodities_page.returned_count == commodities.count, records_page.returned_count == records.count,
              commodities.allSatisfy(\.validContract), Set(commodities.map(\.id)).count == commodities.count,
              Set(records.map(\.id)).count == records.count,
              records.allSatisfy({ !$0.id.isEmpty && $0.base == base && $0.quote == quote && !quote.isEmpty && !$0.rate.exact.isEmpty &&
                  $0.source_order >= 0 && $0.segment >= 0 && ($0.value?.isFinite ?? true) &&
                  (date_from.isEmpty || $0.date >= date_from) && (date_to.isEmpty || $0.date <= date_to) }) else {
            throw DecodingError.dataCorrupted(.init(codingPath: decoder.codingPath, debugDescription: "Inconsistent local price projection"))
        }
    }
}

enum HoldingGrouping: String, Codable, Sendable { case account, commodity, costCurrency = "cost_currency" }
enum HoldingSection: String, Codable, Sendable { case groups, lots, events, history }
enum HoldingScope: String, Codable, Sendable { case direct, subtree, assetsAndLiabilities = "assets-and-liabilities" }
enum HoldingInventoryKind: String, Codable, Sendable { case long, short, mixed, plain }
enum HoldingLotOrigin: String, Codable, Sendable { case posting, averageMerge = "average-merge" }
enum ReductionKind: String, Codable, Sendable {
    case sale, reduction, transferOrExchange = "transfer-or-exchange", shortOrMixed = "short-or-mixed"
}
enum HoldingReason: String, Codable, Sendable {
    case known = ""
    case missingCost = "missing-cost", mixedCostCurrencies = "mixed-cost-currencies"
    case missingSalePrice = "missing-sale-price", missingProceedsQuote = "missing-proceeds-quote"
    case negativeSalePrice = "negative-sale-price"
    case transferOrExchange = "transfer-or-exchange", shortPositionOpening = "short-position-opening"
    case unattributedExpenses = "unattributed-expenses", ambiguousTradeFees = "ambiguous-trade-fees"
    case missingFeeOrSettlementQuote = "missing-fee-or-settlement-quote"
    case proceedsNotReconciled = "proceeds-not-reconciled", unsupportedTradeCounterpart = "unsupported-trade-counterpart"
    case missingPrice = "missing-price", missingCostQuote = "missing-cost-quote"
}

struct ProjectionPage: Codable, Sendable {
    let total_count: Int
    let offset: Int
    let limit: Int
    let returned_count: Int
    let has_more: Bool
    let truncated: Bool
    enum CodingKeys: String, CodingKey { case total_count, offset, limit, returned_count, has_more, truncated }
    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        total_count = try values.decode(Int.self, forKey: .total_count)
        offset = try values.decode(Int.self, forKey: .offset)
        limit = try values.decode(Int.self, forKey: .limit)
        returned_count = try values.decode(Int.self, forKey: .returned_count)
        has_more = try values.decode(Bool.self, forKey: .has_more)
        truncated = try values.decode(Bool.self, forKey: .truncated)
        guard total_count >= 0, offset >= 0, limit >= 0, returned_count >= 0,
              returned_count <= limit, returned_count <= total_count - min(offset, total_count),
              has_more == (min(offset, total_count) + returned_count < total_count),
              truncated == (returned_count < total_count) else {
            throw DecodingError.dataCorrupted(.init(codingPath: decoder.codingPath, debugDescription: "Invalid projection pagination"))
        }
    }
}

struct HoldingLot: Codable, Sendable, Identifiable {
    let id: String
    let account: String
    let commodity: String
    let quantity: DisplayAmount
    @RequiredNullable var cost_currency: String?
    @RequiredNullable var cost_number: DisplayAmount?
    @RequiredNullable var cost_basis: DisplayAmount?
    @RequiredNullable var cost_date: String?
    @RequiredNullable var cost_label: String?
    let source: SourceLocation
    let origin: HoldingLotOrigin
    var validCost: Bool {
        if cost_currency == nil { return cost_number == nil && cost_basis == nil && cost_date == nil && cost_label == nil }
        return cost_number != nil && cost_basis != nil
    }
}

struct HoldingGroup: Codable, Sendable, Identifiable {
    let id: String
    let account: String
    let accounts: [String]
    let commodity: String
    @RequiredNullable var cost_currency: String?
    let quantity: DisplayAmount
    @RequiredNullable var cost_basis: DisplayAmount?
    @RequiredNullable var average_cost: DisplayAmount?
    let lots: [HoldingLot]
    let inventory_kind: HoldingInventoryKind
    let cost_valuation: AccountValuation
    let market_valuation: AccountValuation
    @RequiredNullable var unrealized_gain: DisplayAmount?
    let gain_reason: HoldingReason
    let lots_page: ProjectionPage
    var validContract: Bool {
        let costShape = cost_currency == nil ? cost_basis == nil && average_cost == nil : cost_basis != nil && (quantity.exact == "0" ? average_cost == nil : average_cost != nil)
        let gainShape = unrealized_gain == nil ? gain_reason != .known : gain_reason == .known && cost_valuation.complete_total != nil && market_valuation.complete_total != nil
        return costShape && gainShape && lots_page.returned_count == lots.count && lots.allSatisfy(\.validCost)
    }
}

struct AverageCostPoint: Codable, Sendable {
    let date: String
    let quantity: DisplayAmount
    let cost_basis: DisplayAmount
    @RequiredNullable var average_cost: DisplayAmount?
    @RequiredNullable var value: Double?
    let segment: Int
}
struct AverageCostSeries: Codable, Sendable, Identifiable {
    let id: String
    let account: String
    let accounts: [String]
    let commodity: String
    let cost_currency: String
    let points: [AverageCostPoint]
    let points_page: ProjectionPage
}

struct ReductionEvent: Codable, Sendable, Identifiable {
    let id: String
    let date: String
    let account: String
    let commodity: String
    let kind: ReductionKind
    let quantity: DisplayAmount
    let consumed_quantity: DisplayAmount
    @RequiredNullable var cost_currency: String?
    @RequiredNullable var cost_basis: DisplayAmount?
    let price_currency: String
    @RequiredNullable var proceeds: DisplayAmount?
    @RequiredNullable var proceeds_at_cost: DisplayAmount?
    @RequiredNullable var conversion: ValuationQuote?
    @RequiredNullable var gross_gain: DisplayAmount?
    @RequiredNullable var fees: DisplayAmount?
    @RequiredNullable var net_gain: DisplayAmount?
    let gross_reason: HoldingReason
    let net_reason: HoldingReason
    let lots: [HoldingLot]
    let lots_page: ProjectionPage
    let source: SourceLocation
    let narration: String
    let merge_lots: [HoldingLot]
    let merge_lots_page: ProjectionPage
    var validContract: Bool {
        let grossShape = gross_gain == nil ? gross_reason != .known : gross_reason == .known && cost_currency != nil && cost_basis != nil && proceeds_at_cost != nil && kind == .sale
        let netShape = net_gain == nil ? net_reason != .known && fees == nil : net_reason == .known && fees != nil && gross_gain != nil
        return grossShape && netShape && lots_page.returned_count == lots.count && lots.allSatisfy(\.validCost) && merge_lots_page.returned_count == merge_lots.count && merge_lots.allSatisfy(\.validCost)
    }
}

struct HoldingsPayload: Codable, Sendable {
    let events: [ReductionEvent]
    let average_history: [AverageCostSeries]
    let date_from: String
    let date_to: String
    let account: String
    let scope: HoldingScope
    let groups: [HoldingGroup]
    let currency: String
    let valuation: String
    let group_by: HoldingGrouping
    let section: HoldingSection
    let groups_page: ProjectionPage
    let events_page: ProjectionPage
    let history_page: ProjectionPage
    let source_files: [String: String]
    enum CodingKeys: String, CodingKey {
        case events, average_history, date_from, date_to, account, scope, groups, currency, valuation, group_by, section, groups_page, events_page, history_page, source_files
    }
    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        events = try values.decode([ReductionEvent].self, forKey: .events)
        average_history = try values.decode([AverageCostSeries].self, forKey: .average_history)
        date_from = try values.decode(String.self, forKey: .date_from)
        date_to = try values.decode(String.self, forKey: .date_to)
        account = try values.decode(String.self, forKey: .account)
        scope = try values.decode(HoldingScope.self, forKey: .scope)
        groups = try values.decode([HoldingGroup].self, forKey: .groups)
        currency = try values.decode(String.self, forKey: .currency)
        valuation = try values.decode(String.self, forKey: .valuation)
        group_by = try values.decode(HoldingGrouping.self, forKey: .group_by)
        section = try values.decode(HoldingSection.self, forKey: .section)
        groups_page = try values.decode(ProjectionPage.self, forKey: .groups_page)
        events_page = try values.decode(ProjectionPage.self, forKey: .events_page)
        history_page = try values.decode(ProjectionPage.self, forKey: .history_page)
        source_files = try values.decode([String: String].self, forKey: .source_files)
        guard ["at-cost", "units", "market-value"].contains(valuation),
              groups_page.returned_count == groups.count, events_page.returned_count == events.count, history_page.returned_count == average_history.count,
              groups.allSatisfy(\.validContract), events.allSatisfy(\.validContract),
              average_history.allSatisfy({ $0.points_page.returned_count == $0.points.count && $0.points.allSatisfy({ $0.segment >= 0 && ($0.quantity.exact == "0" ? $0.average_cost == nil : $0.average_cost != nil) && ($0.value?.isFinite ?? true) }) }) else {
            throw DecodingError.dataCorrupted(.init(codingPath: decoder.codingPath, debugDescription: "Inconsistent holdings projection"))
        }
    }

    func source(for quote: ValuationQuote) -> SourceLocation? {
        guard let path = source_files[String(quote.source.file)], !path.isEmpty else { return nil }
        return SourceLocation(path: path, line: quote.source.start_line, column: quote.source.start_column, end_line: quote.source.end_line, end_column: quote.source.end_column)
    }
}

struct AccountLifecycleSource: Codable, Sendable, Identifiable {
    var id: String { kind + ":" + source.path + ":" + String(source.line) }
    let kind: String
    let date: String
    let source: SourceLocation
}

struct AccountHistorySeries: Codable, Sendable, Identifiable {
    var id: String { currency }
    let currency: String
    let mode: String
    let points: [HistoryPoint]
}

struct AccountDetailPayload: Codable, Sendable {
    let account: String
    let scope: String
    let explicit: Bool
    let opened: String
    let closed: String?
    let lifecycle: String
    let booking: String
    let currencies: [String]
    let date_from: String
    let date_to: String
    let valuation: String
    let direct_amounts: [String: DisplayAmount]
    let subtree_amounts: [String: DisplayAmount]
    let direct_valuation: AccountValuation
    let subtree_valuation: AccountValuation
    let period_change: [String: DisplayAmount]
    let period_valuation: AccountValuation
    let sources: [AccountLifecycleSource]
    let history: [AccountHistorySeries]

    enum CodingKeys: String, CodingKey {
        case account, scope, explicit, opened, closed, lifecycle, booking, currencies, date_from, date_to, valuation
        case direct_amounts, subtree_amounts, direct_valuation, subtree_valuation, period_change, period_valuation, sources, history
    }
    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        guard values.contains(.closed) else {
            throw DecodingError.keyNotFound(CodingKeys.closed, .init(codingPath: decoder.codingPath, debugDescription: "Missing nullable closing date"))
        }
        account = try values.decode(String.self, forKey: .account)
        scope = try values.decode(String.self, forKey: .scope)
        explicit = try values.decode(Bool.self, forKey: .explicit)
        opened = try values.decode(String.self, forKey: .opened)
        closed = try values.decodeIfPresent(String.self, forKey: .closed)
        lifecycle = try values.decode(String.self, forKey: .lifecycle)
        booking = try values.decode(String.self, forKey: .booking)
        currencies = try values.decode([String].self, forKey: .currencies)
        date_from = try values.decode(String.self, forKey: .date_from)
        date_to = try values.decode(String.self, forKey: .date_to)
        valuation = try values.decode(String.self, forKey: .valuation)
        direct_amounts = try values.decode([String: DisplayAmount].self, forKey: .direct_amounts)
        subtree_amounts = try values.decode([String: DisplayAmount].self, forKey: .subtree_amounts)
        direct_valuation = try values.decode(AccountValuation.self, forKey: .direct_valuation)
        subtree_valuation = try values.decode(AccountValuation.self, forKey: .subtree_valuation)
        period_change = try values.decode([String: DisplayAmount].self, forKey: .period_change)
        period_valuation = try values.decode(AccountValuation.self, forKey: .period_valuation)
        sources = try values.decode([AccountLifecycleSource].self, forKey: .sources)
        history = try values.decode([AccountHistorySeries].self, forKey: .history)
        guard ["direct", "subtree"].contains(scope), ["open", "closed", "not-opened", "implicit"].contains(lifecycle) else {
            throw DecodingError.dataCorrupted(.init(codingPath: decoder.codingPath, debugDescription: "Unknown account scope or lifecycle"))
        }
    }
}

struct ProfileSummary: Codable, Sendable {
    let semantics_profile: String
    let evaluation_revision: String
    let snapshot_id: String
    let valid: Bool
    let entry_count: Int
    let error_count: Int
    let error_codes: [String: Int]
    let active_extensions: [String]
}

struct MigrationComparison: Codable, Sendable {
    let input_identity: String
    let legacy: ProfileSummary
    let v3: ProfileSummary
    let change_counts: [String: Int]
    let details: [MigrationChange]?
}

struct MigrationChange: Codable, Sendable, Identifiable {
    let dimension: String
    let key: String
    let before: JSONValue
    let after: JSONValue
    var id: String { dimension + key }
}

indirect enum JSONValue: Codable, Sendable {
    case null, string(String), number(Double), bool(Bool), array([JSONValue]), object([String: JSONValue])
    init(from decoder: Decoder) throws {
        let value = try decoder.singleValueContainer()
        if value.decodeNil() { self = .null }
        else if let v = try? value.decode(String.self) { self = .string(v) }
        else if let v = try? value.decode(Bool.self) { self = .bool(v) }
        else if let v = try? value.decode(Double.self) { self = .number(v) }
        else if let v = try? value.decode([JSONValue].self) { self = .array(v) }
        else { self = .object(try value.decode([String: JSONValue].self)) }
    }
    func encode(to encoder: Encoder) throws {
        var value = encoder.singleValueContainer()
        switch self {
        case .null: try value.encodeNil()
        case .string(let v): try value.encode(v)
        case .number(let v): try value.encode(v)
        case .bool(let v): try value.encode(v)
        case .array(let v): try value.encode(v)
        case .object(let v): try value.encode(v)
        }
    }
    var text: String {
        if case .string(let v) = self { return v }
        let encoder = JSONEncoder(); encoder.outputFormatting = [.sortedKeys]
        return (try? encoder.encode(self)).map { String(decoding: $0, as: UTF8.self) } ?? "—"
    }
}

struct SemanticsPreview: Identifiable {
    let profile: String
    let comparison: MigrationComparison
    var id: String { profile + comparison.input_identity }
}

struct OpenLedgerResult: Codable, Sendable {
    let handle: UInt64
    let state: LedgerState
}

struct PollResult: Codable, Sendable {
    let changed: Bool
    let state: LedgerState
}

struct DisplayAmount: Codable, Sendable {
    let exact: String
    let display: String
    let approximate: Bool
    var label: String { (approximate ? "≈" : "") + display }
}

struct SourceLocation: Codable, Sendable {
    let path: String
    let line: Int
    let column: Int
    var end_line: Int? = nil
    var end_column: Int? = nil
}

struct CostComponent: Codable, Sendable {
    let kind: String
    let text: String
    let number: DisplayAmount?
    let currency: String?
}

struct PostingCost: Codable, Sendable {
    let total: Bool
    let display: String
    let components: [CostComponent]
}

struct PostingPrice: Codable, Sendable {
    let total: Bool
    let number: DisplayAmount
    let currency: String
}

struct RepairExample: Codable, Sendable {
    let before: String
    let after: String
    let note: String
}

struct RepairGuide: Codable, Sendable {
    let what: String
    let why: String
    let inspect: [String]
    let safe_steps: [String]
    let example: RepairExample
    let revalidate: String
}

struct RelatedLocation: Codable, Sendable {
    let source: SourceLocation
    let message: String
}

struct LedgerDiagnostic: Codable, Sendable {
    let code: String
    let severity: String
    let message: String
    let source: SourceLocation
    let related: [RelatedLocation]
    let guide: RepairGuide?
}

// Summary payload returned from Go Core
struct LedgerSummary: Codable, Sendable {
    var valid: Bool
    var entry_path: String?
    var title: String?
    var operating_currencies: [String]?
    var accounts: [String]
    var entries_count: Int
    var error_count: Int
    var errors: [String]
    var transactions_count: Int?
    var diagnostics: [LedgerDiagnostic]?
    var currencies: [String]?
}

struct MissingValuation: Codable, Sendable {
    let commodity: String
    let reason: String
}

struct QuoteSpan: Codable, Sendable {
    let file: UInt64
    let start: Int
    let end: Int
    let start_line: Int
    let start_column: Int
    let end_line: Int
    let end_column: Int
}

struct ValuationQuote: Codable, Sendable {
    let base: String
    let quote: String
    let date: String
    let rate: DisplayAmount
    let inverse: Bool
    let source: QuoteSpan
}

struct ValuationStatus: Codable, Sendable {
    let mode: String
    let target_currency: String
    let as_of: String
    let completeness: String
    let covered_item_count: Int
    let missing_item_count: Int
    let missing: [MissingValuation]
    let quotes: [ValuationQuote]
}

struct AccountValuation: Codable, Sendable {
    let mode: String
    let target_currency: String
    let as_of: String
    let completeness: String
    let covered_item_count: Int
    let missing_item_count: Int
    let missing: [MissingValuation]
    let quotes: [ValuationQuote]
    let valued_portion: DisplayAmount?
    let complete_total: DisplayAmount?

    enum CodingKeys: String, CodingKey {
        case mode, target_currency, as_of, completeness, covered_item_count, missing_item_count, missing, quotes, valued_portion, complete_total
    }
    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        for field in [CodingKeys.valued_portion, .complete_total] {
            guard values.contains(field) else {
                throw DecodingError.keyNotFound(field, .init(codingPath: decoder.codingPath, debugDescription: "Required nullable valuation field missing"))
            }
        }
        mode = try values.decode(String.self, forKey: .mode)
        target_currency = try values.decode(String.self, forKey: .target_currency)
        as_of = try values.decode(String.self, forKey: .as_of)
        completeness = try values.decode(String.self, forKey: .completeness)
        covered_item_count = try values.decode(Int.self, forKey: .covered_item_count)
        missing_item_count = try values.decode(Int.self, forKey: .missing_item_count)
        missing = try values.decode([MissingValuation].self, forKey: .missing)
        quotes = try values.decode([ValuationQuote].self, forKey: .quotes)
        valued_portion = try values.decodeIfPresent(DisplayAmount.self, forKey: .valued_portion)
        complete_total = try values.decodeIfPresent(DisplayAmount.self, forKey: .complete_total)
        let knownState = ["complete", "partial", "unavailable", "empty"].contains(completeness)
        let hasInvalidTotal = completeness != "complete" && complete_total != nil
        let incompleteComplete = mode == "converted" && completeness == "complete" && complete_total == nil
        let inconsistentTotal = complete_total != nil && complete_total?.exact != valued_portion?.exact
        let inconsistentCoverage = completeness == "complete" && missing_item_count != 0
        guard ["native", "converted"].contains(mode), knownState,
              covered_item_count >= 0, missing_item_count >= 0,
              !hasInvalidTotal, !incompleteComplete, !inconsistentTotal, !inconsistentCoverage else {
            throw DecodingError.dataCorrupted(.init(codingPath: decoder.codingPath, debugDescription: "Invalid valuation contract"))
        }
    }
}

// Tree report response
struct TreeReportPayload: Codable, Sendable {
    var valuation_status: ValuationStatus? = nil
    var currency: String
    var date_from: String
    var date_to: String
    var availability: String
    var report_type: String
    var operating_currencies: [String]?
    var root_nodes: [AccountTreeNode]
    var error: String?
}

// Hierarchical account node directly matching internal/report.AccountTreeNode
final class AccountTreeNode: Codable, Sendable, Identifiable, ObservableObject {
    var id: String { full_name }
    let name: String
    let full_name: String
    let depth: int_fast32_t
    let is_leaf: Bool
    let explicit: Bool
    let direct_amounts: [String: DisplayAmount]
    let subtree_amounts: [String: DisplayAmount]
    let direct_balances: [String: String]
    let subtree_balances: [String: String]
    let children: [AccountTreeNode]
    let direct_valuation: AccountValuation?
    let subtree_valuation: AccountValuation?

    init(name: String, fullName: String, depth: int_fast32_t, isLeaf: Bool, explicit: Bool, directBalances: [String: String], subtreeBalances: [String: String], directAmounts: [String: DisplayAmount], subtreeAmounts: [String: DisplayAmount], children: [AccountTreeNode], directValuation: AccountValuation? = nil, subtreeValuation: AccountValuation? = nil) {
        self.direct_valuation = directValuation
        self.subtree_valuation = subtreeValuation
        self.direct_amounts = directAmounts
        self.subtree_amounts = subtreeAmounts
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
struct JournalPosting: Codable, Sendable, Identifiable {
    var id: String { "\(account)_\(units_number)_\(units_currency)" }
    let account: String
    let units_number: String
    let units_currency: String
    let cost: PostingCost?
    let price: PostingPrice?
    let source: SourceLocation
    let amount: DisplayAmount?
}

struct JournalTransaction: Codable, Sendable, Identifiable {
    let id: String
    let date: String
    let flag: String
    let payee: String
    let narration: String
    let tags: [String]
    let links: [String]
    let postings: [JournalPosting]
    let source: SourceLocation
}

struct JournalPayload: Codable, Sendable {
    let total_count: Int
    let offset: Int
    let limit: Int
    let has_more: Bool
    let date_from: String
    let date_to: String
    let transactions: [JournalTransaction]
    let error: String?
}

// Chart models for Swift Charts
struct TrendPoint: Codable, Sendable, Identifiable {
    var id: String { date }
    let date: String
    let value: Double
    let currency: String
    let amount: DisplayAmount?
}

struct CashFlowBar: Codable, Sendable, Identifiable {
    var id: String { month }
    let month: String
    let income: Double
    let expenses: Double
    let net: Double
    let currency: String
    let income_amount: DisplayAmount?
    let expenses_amount: DisplayAmount?
    let net_amount: DisplayAmount?
}

struct HistoryPoint: Codable, Sendable, Identifiable {
    var id: String { date }
    let date: String
    let month: String
    let segment: Int
    let value: Double?
    let valuation: AccountValuation
}

struct HistoryCashFlow: Codable, Sendable, Identifiable {
    var id: String { month }
    let month: String
    let date: String
    let income: AccountValuation
    let expenses: AccountValuation
    let net: AccountValuation
    let income_value: Double?
    let expenses_value: Double?
}

struct CurrencyHistorySeries: Codable, Sendable, Identifiable {
    var id: String { currency }
    let currency: String
    let mode: String
    let valuation: String
    let net_worth_points: [HistoryPoint]
    let monthly_cash_flows: [HistoryCashFlow]
    let income_total: DisplayAmount?
    let expenses_total: DisplayAmount?
    let net_total: DisplayAmount?
    let net_worth_availability: String
    let cash_flow_availability: String
}

struct HistoricalTrendsPayload: Codable, Sendable {
    var currency_series: [CurrencyHistorySeries]? = nil
    let income_amount: DisplayAmount
    let expenses_amount: DisplayAmount
    let net_amount: DisplayAmount
    let date_from: String
    let date_to: String
    let operating_currency: String
    let net_worth_availability: String?
    let cash_flow_availability: String?
    let net_worth_points: [TrendPoint]
    let monthly_cash_flows: [CashFlowBar]
    let error: String?
}

struct MigrationPresentation {
    let key: String
    let before: String
    let after: String
    let detail: String?
}

extension MigrationChange {
    func presentation(locale: String) -> MigrationPresentation {
        let chinese = locale == "zh-CN"
        func text(_ zh: String, _ en: String) -> String { chinese ? zh : en }
        let decoded = try? JSONDecoder().decode(JSONValue.self, from: Data(key.utf8))
        var title = key.replacingOccurrences(of: "\u{0}", with: " · ")
        var detail: String?
        if dimension == "entries" {
            let date = decoded?["Date"]["Raw"].plain ?? ""
            let narration = decoded?["Narration"].plain ?? decoded?["Account"].plain ?? ""
            let line = decoded?["At"]["start_line"].plain ?? ""
            title = [date, narration, line.isEmpty ? "" : text("行 ", "Line ") + line].filter { !$0.isEmpty }.joined(separator: " · ")
            if title.isEmpty {title = text("指令变化", "Directive change")}
            detail = decoded?["Postings"].items.map { posting in
                var value = [posting["Flag"].plain, posting["Account"].plain, posting["Units"]["Number"]["Raw"].plain, posting["Units"]["Currency"].plain].compactMap { $0 }.filter { !$0.isEmpty }.joined(separator: " ")
                let costs = posting["Cost"]["Components"].items.compactMap { component -> String? in
                    switch component["Kind"].plain {
                    case "amount": return [component["Amount"]["Number"]["Raw"].plain, component["Amount"]["Currency"].plain].compactMap { $0 }.joined(separator: " ")
                    case "date": return component["Date"]["Raw"].plain
                    case "string": return component["String"].plain.map { "\"" + $0 + "\"" }
                    default: return nil
                    }
                }
                if !costs.isEmpty {value += " {" + costs.joined(separator: ", ") + "}"}
                if let quote = posting["Price"]["Amount"]["Number"]["Raw"].plain, let currency = posting["Price"]["Amount"]["Currency"].plain {
                    value += (posting["Price"]["Total"].plain == "true" ? " @@ " : " @ ") + quote + " " + currency
                }
                return value
            }.joined(separator: "\n")
        } else if dimension == "lots", let fields = decoded?.items, fields.count >= 6 {
            title = [fields[0].plain, fields[1].plain, text("单位成本 ", "Unit cost ") + (fields[2].plain ?? "—") + " " + (fields[3].plain ?? ""), fields[4].plain, fields[5].plain].compactMap { $0 }.filter { !$0.isEmpty }.joined(separator: " · ")
        } else if dimension == "diagnostics" {
            title = [decoded?["code"].plain, decoded?["message"].plain].compactMap { $0 }.joined(separator: " · ")
        }
        func description(_ value: JSONValue) -> String {
            if case .null = value {return text("不存在", "Absent")}
            if dimension == "entries" || dimension == "diagnostics" {return (value.plain ?? "—") + text(" 条匹配记录", " matching record(s)")}
            if dimension == "accounts" {
                return [text("启用 ", "Opened ") + (value["Opened"].plain ?? "—"), text("关闭 ", "Closed ") + (value["Closed"]["Raw"].plain ?? "—"), text("规则 ", "Booking ") + (value["Booking"].plain.flatMap { $0.isEmpty ? nil : $0 } ?? text("隐含", "Implicit")), text("币种 ", "Currencies ") + value["Currencies"].items.compactMap { $0.plain }.joined(separator: ", ")].joined(separator: " · ")
            }
            return value.plain ?? "—"
        }
        return MigrationPresentation(key:title, before:description(before), after:description(after), detail:detail)
    }
}

extension JSONValue {
    subscript(_ field: String) -> JSONValue {
        if case .object(let values) = self {return values[field] ?? .null}
        return .null
    }
    var items: [JSONValue] {
        if case .array(let values) = self {return values}
        return []
    }
    var plain: String? {
        switch self {
        case .string(let value): return value
        case .number(let value): return String(format: "%.0f", value)
        case .bool(let value): return value ? "true" : "false"
        default: return nil
        }
    }
}

// Native queries retain ordered schema and positional cells, including duplicate titles.
struct QueryRequest: Codable, Sendable {
    let snapshot_id: String
    let request_id: UInt64
    let text: String
}

struct QueryContext: Codable, Sendable {
    let snapshot_id: String
    let semantics_profile: String
    let evaluation_revision: String
    let request_id: UInt64
    let text: String
    let fingerprint: String
}

struct QueryExportRequest: Codable, Sendable {
    let snapshot_id: String
    let request_id: UInt64
    let fingerprint: String
    let format: String
}

struct QueryExportCapture: Codable, Sendable {
    let context: QueryContext
    let format: String
    let content: Data
    var protected_paths: [String] = []
}

struct ExportProtection: Codable, Sendable {
    let revision: UInt64
    let snapshot_id: String
    let protected_paths: [String]
}

struct QueryByteRange: Codable, Sendable {
    let start_byte: Int
    let end_byte: Int

    func nativeRange(in text: String) -> NSRange? {
        guard start_byte >= 0, end_byte >= start_byte, end_byte <= text.utf8.count else { return nil }
        let bytes = text.utf8
        let lowerByte = bytes.index(bytes.startIndex, offsetBy: start_byte)
        let upperByte = bytes.index(bytes.startIndex, offsetBy: end_byte)
        guard let lower = String.Index(lowerByte, within: text), let upper = String.Index(upperByte, within: text) else { return nil }
        return NSRange(lower..<upper, in: text)
    }
}

struct QueryFailure: Codable, Sendable {
    let code: String
    let message: String
    let phase: String
    let range: QueryByteRange?
    enum CodingKeys: String, CodingKey { case code, message, phase, range }
    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        guard values.contains(.range) else { throw queryContractError(decoder, "Missing query error range") }
        code = try values.decode(String.self, forKey: .code)
        message = try values.decode(String.self, forKey: .message)
        phase = try values.decode(String.self, forKey: .phase)
        range = try values.decodeIfPresent(QueryByteRange.self, forKey: .range)
    }
    func encode(to encoder: Encoder) throws {
        var values = encoder.container(keyedBy: CodingKeys.self)
        try values.encode(code, forKey: .code); try values.encode(message, forKey: .message)
        try values.encode(phase, forKey: .phase); try values.encode(range, forKey: .range)
    }
}

struct QueryColumn: Codable, Sendable, Identifiable {
    let id: String
    let name: String
    let type: String
    let nullable: Bool
}

private func queryContractError(_ decoder: Decoder, _ message: String) -> DecodingError {
    .dataCorrupted(.init(codingPath: decoder.codingPath, debugDescription: message))
}

private func validateQueryDecimal(_ value: DisplayAmount, _ decoder: Decoder) throws {
    guard value.exact.range(of: #"^-?(?:[0-9]+(?:\.[0-9]+)?|[0-9]+/[0-9]+)$"#, options: .regularExpression) != nil,
          !value.exact.hasSuffix("/0") else { throw queryContractError(decoder, "Invalid exact query number") }
}

struct QueryAmount: Codable, Sendable {
    let number: DisplayAmount
    let currency: String
    enum CodingKeys: String, CodingKey { case number, currency }
    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        number = try values.decode(DisplayAmount.self, forKey: .number)
        currency = try values.decode(String.self, forKey: .currency)
        try validateQueryDecimal(number, decoder)
        guard !currency.isEmpty else { throw queryContractError(decoder, "Missing amount currency") }
    }
}

struct QueryCost: Codable, Sendable {
    let number: DisplayAmount
    let currency: String
    let date: String?
    let label: String?
    enum CodingKeys: String, CodingKey { case number, currency, date, label }
    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        guard values.contains(.date), values.contains(.label) else { throw queryContractError(decoder, "Missing nullable query cost fields") }
        number = try values.decode(DisplayAmount.self, forKey: .number)
        currency = try values.decode(String.self, forKey: .currency)
        date = try values.decodeIfPresent(String.self, forKey: .date)
        label = try values.decodeIfPresent(String.self, forKey: .label)
        try validateQueryDecimal(number, decoder)
        guard !currency.isEmpty else { throw queryContractError(decoder, "Missing cost currency") }
    }
    func encode(to encoder: Encoder) throws {
        var values = encoder.container(keyedBy: CodingKeys.self)
        try values.encode(number, forKey: .number); try values.encode(currency, forKey: .currency)
        try values.encode(date, forKey: .date); try values.encode(label, forKey: .label)
    }
}

struct QueryPosition: Codable, Sendable {
    let units: QueryAmount
    let cost: QueryCost?
    enum CodingKeys: String, CodingKey { case units, cost }
    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        guard values.contains(.cost) else { throw queryContractError(decoder, "Missing nullable query position cost") }
        units = try values.decode(QueryAmount.self, forKey: .units)
        cost = try values.decodeIfPresent(QueryCost.self, forKey: .cost)
    }
    func encode(to encoder: Encoder) throws {
        var values = encoder.container(keyedBy: CodingKeys.self)
        try values.encode(units, forKey: .units); try values.encode(cost, forKey: .cost)
    }
}

enum QueryValue: Sendable {
    case null, text(String), integer(String), boolean(Bool), decimal(DisplayAmount)
    case amount(QueryAmount), position(QueryPosition), inventory([QueryPosition])
}

struct QueryCell: Codable, Sendable {
    let type: String
    let value: QueryValue
    enum CodingKeys: String, CodingKey { case type, value }
    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        type = try values.decode(String.self, forKey: .type)
        switch type {
        case "null":
            guard !values.contains(.value) else { throw queryContractError(decoder, "NULL cell must not contain a value") }
            value = .null
        case "string", "date": value = .text(try values.decode(String.self, forKey: .value))
        case "integer":
            let integer = try values.decode(String.self, forKey: .value)
            guard integer.range(of: #"^-?[0-9]+$"#, options: .regularExpression) != nil else { throw queryContractError(decoder, "Invalid query integer") }
            value = .integer(integer)
        case "bool": value = .boolean(try values.decode(Bool.self, forKey: .value))
        case "decimal":
            let decimal = try values.decode(DisplayAmount.self, forKey: .value)
            try validateQueryDecimal(decimal, decoder); value = .decimal(decimal)
        case "amount": value = .amount(try values.decode(QueryAmount.self, forKey: .value))
        case "position": value = .position(try values.decode(QueryPosition.self, forKey: .value))
        case "inventory": value = .inventory(try values.decode([QueryPosition].self, forKey: .value))
        default: throw queryContractError(decoder, "Unknown query cell type")
        }
    }
    func encode(to encoder: Encoder) throws {
        var values = encoder.container(keyedBy: CodingKeys.self)
        try values.encode(type, forKey: .type)
        switch value {
        case .null: break
        case .text(let text), .integer(let text): try values.encode(text, forKey: .value)
        case .boolean(let bool): try values.encode(bool, forKey: .value)
        case .decimal(let decimal): try values.encode(decimal, forKey: .value)
        case .amount(let amount): try values.encode(amount, forKey: .value)
        case .position(let position): try values.encode(position, forKey: .value)
        case .inventory(let inventory): try values.encode(inventory, forKey: .value)
        }
    }
}

struct QueryTypedResult: Codable, Sendable {
    let contract: String
    let columns: [QueryColumn]
    let rows: [[QueryCell]]
    enum CodingKeys: String, CodingKey { case contract, columns, rows }
    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        contract = try values.decode(String.self, forKey: .contract)
        columns = try values.decode([QueryColumn].self, forKey: .columns)
        rows = try values.decode([[QueryCell]].self, forKey: .rows)
        let types: Set<String> = ["null", "string", "date", "integer", "bool", "decimal", "amount", "position", "inventory", "object"]
        guard contract == "beanquery-0.2.0-typed-v1", Set(columns.map(\.id)).count == columns.count,
              columns.allSatisfy({ !$0.id.isEmpty && types.contains($0.type) }),
              rows.allSatisfy({ row in
                  row.count == columns.count && zip(row, columns).allSatisfy { cell, column in
                      cell.type == "null" ? column.nullable : (column.type == "object" || cell.type == column.type)
                  }
              }) else { throw queryContractError(decoder, "Invalid typed query schema or cells") }
    }
}

struct QueryExecution: Codable, Sendable {
    let result: QueryTypedResult
    let returned_rows: Int
    let row_limit: Int
    let truncated: Bool
    enum CodingKeys: String, CodingKey { case result, returned_rows, row_limit, truncated }
    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        result = try values.decode(QueryTypedResult.self, forKey: .result)
        returned_rows = try values.decode(Int.self, forKey: .returned_rows)
        row_limit = try values.decode(Int.self, forKey: .row_limit)
        truncated = try values.decode(Bool.self, forKey: .truncated)
        guard (1...10000).contains(row_limit), returned_rows == result.rows.count, returned_rows <= row_limit,
              !truncated || returned_rows == row_limit else { throw queryContractError(decoder, "Invalid query result budget metadata") }
    }
}

struct QueryResponse: Codable, Sendable {
    let context: QueryContext
    let execution: QueryExecution?
    let error: QueryFailure?
    enum CodingKeys: String, CodingKey { case context, execution, error }
    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        guard values.contains(.execution), values.contains(.error) else { throw queryContractError(decoder, "Missing nullable query execution fields") }
        context = try values.decode(QueryContext.self, forKey: .context)
        execution = try values.decodeIfPresent(QueryExecution.self, forKey: .execution)
        error = try values.decodeIfPresent(QueryFailure.self, forKey: .error)
        guard (execution == nil) != (error == nil), context.request_id > 0, !context.snapshot_id.isEmpty,
              ["legacy", "v3"].contains(context.semantics_profile), !context.evaluation_revision.isEmpty,
              context.fingerprint.range(of: #"^[0-9a-f]{64}$"#, options: .regularExpression) != nil else {
            throw queryContractError(decoder, "Invalid query execution context")
        }
        if let range = error?.range {
            guard range.start_byte >= 0, range.end_byte >= range.start_byte, range.end_byte <= context.text.utf8.count, range.nativeRange(in: context.text) != nil else {
                throw queryContractError(decoder, "Invalid query error byte offsets")
            }
        }
    }
    func encode(to encoder: Encoder) throws {
        var values = encoder.container(keyedBy: CodingKeys.self)
        try values.encode(context, forKey: .context)
        try values.encode(execution, forKey: .execution); try values.encode(error, forKey: .error)
    }
}

struct SavedQuery: Codable, Sendable, Identifiable {
    let id: String
    let date: String
    let name: String
    let text: String
    let source: SourceLocation
}

struct QueryPreset: Codable, Sendable, Identifiable {
    let id: String
    let name_zh: String
    let name_en: String
    let text: String
}

struct QueryPresetPeriod: Codable, Sendable {
    let date_from: String
    let date_to: String
    let presets: [QueryPreset]
}

private func validPresetDate(_ value: String) -> Bool {
    if value.isEmpty { return true }
    let bytes = Array(value.utf8)
    guard bytes.count == 10, bytes[4] == 45, bytes[7] == 45,
          bytes.enumerated().allSatisfy({ $0.offset == 4 || $0.offset == 7 || (48...57).contains($0.element) }),
          let year = Int(value.prefix(4)), let month = Int(value.dropFirst(5).prefix(2)), let day = Int(value.suffix(2)),
          (1...12).contains(month) else { return false }
    let leap = year % 400 == 0 || (year % 4 == 0 && year % 100 != 0)
    let days = [31, leap ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31]
    return day >= 1 && day <= days[month - 1]
}

struct QueryCatalogue: Codable, Sendable {
    let snapshot_id: String
    let queries: [SavedQuery]
    let presets: [QueryPreset]
    let period: QueryPresetPeriod?
    let page: ProjectionPage
    enum CodingKeys: String, CodingKey { case snapshot_id, queries, presets, period, page }
    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        snapshot_id = try values.decode(String.self, forKey: .snapshot_id)
        queries = try values.decode([SavedQuery].self, forKey: .queries)
        presets = try values.decodeIfPresent([QueryPreset].self, forKey: .presets) ?? []
        period = try values.decodeIfPresent(QueryPresetPeriod.self, forKey: .period)
        if let period {
            guard validPresetDate(period.date_from), validPresetDate(period.date_to),
                  !period.date_from.isEmpty || !period.date_to.isEmpty,
                  period.date_from.isEmpty || period.date_to.isEmpty || period.date_from <= period.date_to,
                  period.presets.map(\.id) == presets.map(\.id),
                  period.presets.allSatisfy({ !$0.text.isEmpty && $0.text.utf8.count <= 65536 && !$0.name_zh.isEmpty && !$0.name_en.isEmpty }) else {
                throw queryContractError(decoder, "Invalid preset period")
            }
        }
        page = try values.decode(ProjectionPage.self, forKey: .page)
        guard presets.count <= 32, Set(presets.map(\.id)).count == presets.count,
              presets.allSatisfy({ !$0.id.isEmpty && !$0.name_zh.isEmpty && !$0.name_en.isEmpty && !$0.text.isEmpty && $0.text.utf8.count <= 65536 }),
              !snapshot_id.isEmpty, page.limit > 0, page.limit <= 500,
              queries.count == page.returned_count, Set(queries.map(\.id)).count == queries.count,
              queries.allSatisfy({ !$0.id.isEmpty && !$0.source.path.isEmpty && $0.source.line > 0 && $0.source.column > 0 }) else {
            throw queryContractError(decoder, "Invalid saved query catalogue")
        }
    }
}

indirect enum DirectiveContent: Sendable {
    case null, text(String), boolean(Bool), decimal(DisplayAmount), amount(QueryAmount)
    case list([DirectiveValue]), map([DirectiveMetadata])
}

struct DirectiveValue: Codable, Sendable {
    let type: String
    let value: DirectiveContent
    enum CodingKeys: String, CodingKey { case type, value }
    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        type = try values.decode(String.self, forKey: .type)
        switch type {
        case "null":
            guard !values.contains(.value) else { throw queryContractError(decoder, "NULL directive value contains data") }
            value = .null
        case "string", "account", "currency", "tag", "link", "date":
            let text = try values.decode(String.self, forKey: .value)
            if type == "date", !validPresetDate(text) || text.isEmpty { throw queryContractError(decoder, "Invalid directive date") }
            value = .text(text)
        case "bool": value = .boolean(try values.decode(Bool.self, forKey: .value))
        case "decimal":
            let number = try values.decode(DisplayAmount.self, forKey: .value)
            try validateQueryDecimal(number, decoder); value = .decimal(number)
        case "amount": value = .amount(try values.decode(QueryAmount.self, forKey: .value))
        case "list": value = .list(try values.decode([DirectiveValue].self, forKey: .value))
        case "map": value = .map(try values.decode([DirectiveMetadata].self, forKey: .value))
        default: throw queryContractError(decoder, "Unknown directive value type")
        }
    }
    func encode(to encoder: Encoder) throws {
        var values = encoder.container(keyedBy: CodingKeys.self)
        try values.encode(type, forKey: .type)
        switch value {
        case .null: break
        case .text(let text): try values.encode(text, forKey: .value)
        case .boolean(let boolean): try values.encode(boolean, forKey: .value)
        case .decimal(let number): try values.encode(number, forKey: .value)
        case .amount(let amount): try values.encode(amount, forKey: .value)
        case .list(let list): try values.encode(list, forKey: .value)
        case .map(let fields): try values.encode(fields, forKey: .value)
        }
    }
    var exactText: String {
        switch value {
        case .null: return "NULL"
        case .text(let text): return text.isEmpty ? "\"\"" : (type == "tag" ? "#" : type == "link" ? "^" : "") + text
        case .boolean(let boolean): return boolean ? "TRUE" : "FALSE"
        case .decimal(let number): return number.exact
        case .amount(let amount): return amount.number.exact + " " + amount.currency
        case .list(let items): return "[" + items.map(\.exactText).joined(separator: ", ") + "]"
        case .map(let fields): return "{" + fields.map { $0.key + ": " + $0.value.exactText }.joined(separator: ", ") + "}"
        }
    }
}

struct DirectiveField: Codable, Sendable { let name: String; let value: DirectiveValue }
struct DirectiveMetadata: Codable, Sendable { let key: String; let value: DirectiveValue; let source: SourceLocation }
struct DirectivePosting: Codable, Sendable { let posting: JournalPosting; let metadata: [DirectiveMetadata] }

struct Attachment: Codable, Sendable, Identifiable {
    let id: String
    let filename: String
    let path: String
    let status: String
}
struct DocumentAction: Codable, Sendable { let id: String; let path: String }

struct DirectiveRecord: Codable, Sendable, Identifiable {
    let attachments: [Attachment]
    let id: String
    let kind: String
    let source_kind: String
    @RequiredNullable var date: String?
    let generated: Bool
    @RequiredNullable var origin_id: String?
    let source: SourceLocation
    let fields: [DirectiveField]
    let metadata: [DirectiveMetadata]
    let tags: [String]
    let links: [String]
    let postings: [DirectivePosting]
}

struct DirectivesPayload: Codable, Sendable {
    let snapshot_id: String
    let mode: String
    let date_from: String
    let date_to: String
    let page: ProjectionPage
    let records: [DirectiveRecord]
    enum CodingKeys: String, CodingKey { case snapshot_id, mode, date_from, date_to, page, records }
    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        snapshot_id = try values.decode(String.self, forKey: .snapshot_id)
        mode = try values.decode(String.self, forKey: .mode)
        date_from = try values.decode(String.self, forKey: .date_from)
        date_to = try values.decode(String.self, forKey: .date_to)
        page = try values.decode(ProjectionPage.self, forKey: .page)
        records = try values.decode([DirectiveRecord].self, forKey: .records)
        let config: Set<String> = ["option", "plugin", "include", "pushtag", "poptag", "pushmeta", "popmeta"]
        let dated: Set<String> = ["open", "close", "commodity", "balance", "pad", "event", "query", "price", "document", "note", "custom", "transaction"]
        guard validPresetDate(date_from), validPresetDate(date_to),
              (date_from.isEmpty || date_to.isEmpty || date_from <= date_to),
              (mode != "config" || (date_from.isEmpty && date_to.isEmpty)),
              !snapshot_id.isEmpty, ["all", "config"].contains(mode), page.limit > 0, page.limit <= 500,
              page.returned_count == records.count, Set(records.map(\.id)).count == records.count,
              records.allSatisfy({ record in
                  !record.id.isEmpty && !record.source.path.isEmpty && record.source.line > 0 && record.source.column > 0 &&
                  (mode == "config" ? config.contains(record.kind) && record.date == nil : dated.contains(record.kind) && record.date != nil && !record.date!.isEmpty && validPresetDate(record.date!)) &&
                  (record.generated ? record.kind == "transaction" && (record.origin_id != nil || record.source_kind.isEmpty) : record.origin_id == nil && record.source_kind == record.kind) &&
                  (record.source_kind.isEmpty || config.union(dated).union(["dialect"]).contains(record.source_kind)) &&
                  (record.kind == "transaction" || record.postings.isEmpty) &&
                  (record.kind == "document" || record.attachments.isEmpty) &&
                  (record.kind != "document" || record.attachments.map(\.filename) == record.fields.filter { $0.name == "filename" }.compactMap { if case .text(let text) = $0.value.value { return text }; return nil }) &&
                  Set(record.attachments.map(\.id)).count == record.attachments.count &&
                  record.attachments.allSatisfy { !$0.id.isEmpty && ["available", "missing", "outside-root", "unsupported-kind", "unreadable"].contains($0.status) && ($0.status != "available" || $0.path.hasPrefix("/")) } && record.validFields
              }) else { throw queryContractError(decoder, "Invalid directive records") }
    }
}

private extension DirectiveRecord {
    var validFields: Bool {
        let shapes: [String: [String: String]] = [
            "option":["key":"string", "value":"string"], "plugin":["module":"string", "config":"string"], "include":["path":"string"],
            "pushtag":["tag":"tag"], "poptag":["tag":"tag"], "pushmeta":["key":"string", "value":"*"], "popmeta":["key":"string"],
            "open":["account":"account", "currency":"currency", "booking":"string"], "close":["account":"account"], "commodity":["currency":"currency"],
            "balance":["account":"account", "amount":"amount", "tolerance":"nullable-decimal"], "pad":["account":"account", "source_account":"account"],
            "event":["event_type":"string", "value":"string"], "query":["name":"string", "query":"string"], "price":["currency":"currency", "amount":"amount"],
            "document":["account":"account", "filename":"string"], "note":["account":"account", "comment":"string"], "custom":["custom_type":"string", "value":"*"],
            "transaction":["flag":"string", "payee":"string", "narration":"string"]]
        guard let shape = shapes[kind] else { return false }
        var counts: [String: Int] = [:]
        for field in fields {
            guard let expected = shape[field.name], expected == "*" || expected == field.value.type || (expected == "nullable-decimal" && ["decimal", "null"].contains(field.value.type)) else { return false }
            counts[field.name, default: 0] += 1
        }
        for name in shape.keys {
            if (kind == "open" && name == "currency") || (kind == "custom" && name == "value") { continue }
            if kind == "document" && name == "filename" { if counts[name, default: 0] == 0 { return false }; continue }
            if counts[name] != 1 { return false }
        }
        return true
    }
}
