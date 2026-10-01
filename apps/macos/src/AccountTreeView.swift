import SwiftUI

final class AccountTreeState: ObservableObject {
    @Published var searchText: String = ""
    @Published var collapsedNodes: Set<String> = []
    @Published var selectedAccount: String?

    func isExpanded(_ fullName: String) -> Bool {
        return !searchText.isEmpty || !collapsedNodes.contains(fullName)
    }

    func toggleExpanded(_ fullName: String) {
        if collapsedNodes.contains(fullName) {
            collapsedNodes.remove(fullName)
        } else {
            collapsedNodes.insert(fullName)
        }
    }
}

struct AccountTreeView: View {
    @Environment(\.ledgerLanguage) private var language
    @FocusState private var searchFocused: Bool
    @FocusState private var listFocused: Bool
    let rootNodes: [AccountTreeNode]
    let operatingCurrencies: [String]
    @Binding var reportType: String
    @ObservedObject var treeState: AccountTreeState
    var openAccount: (String) -> Void = { _ in }

    var filteredNodes: [AccountTreeNode] {
        if treeState.searchText.isEmpty {
            return rootNodes
        }
        return rootNodes.compactMap { filterNode($0, query: treeState.searchText.lowercased()) }
    }

    private var visibleNodes: [AccountTreeNode] {
        func rows(_ nodes: [AccountTreeNode]) -> [AccountTreeNode] {
            nodes.flatMap { node in
                [node] + (treeState.isExpanded(node.full_name) ? rows(node.children) : [])
            }
        }
        return rows(filteredNodes)
    }

    var body: some View {
        VStack(spacing: 0) {
            // Header Bar
            HStack {
                Picker(language.text("报表类型", "Report"), selection: $reportType) {
                    Text(language.text("资产负债表", "Balance Sheet")).tag("balance_sheet")
                    Text(language.text("损益表", "Income Statement")).tag("income_statement")
                }
                .pickerStyle(.segmented)
                .frame(maxWidth: 320)

                Button(language.text("全部展开", "Expand all")) { treeState.collapsedNodes = [] }
                Button(language.text("全部折叠", "Collapse all")) { treeState.collapsedNodes = Set(allNames(rootNodes)) }
                Button(language.text("账户详情", "Account details")) {
                    if let account = treeState.selectedAccount { openAccount(account) }
                }.disabled(treeState.selectedAccount == nil)
                Spacer()

                HStack {
                    Image(systemName: "magnifyingglass")
                        .foregroundColor(.secondary)
                    TextField(language.text("筛选账户…", "Filter accounts…"), text: $treeState.searchText)
                        .focused($searchFocused)
                        .textFieldStyle(.plain)
                        .frame(width: 180)
                    if !treeState.searchText.isEmpty {
                        Button(action: { treeState.searchText = "" }) {
                            Image(systemName: "xmark.circle.fill")
                                .foregroundColor(.secondary)
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(6)
                .background(RoundedRectangle(cornerRadius: 8).fill(.regularMaterial))
            }
            .padding()
            .background(.ultraThinMaterial)

            HStack {
                Text(language.text("搜索只隐藏行；汇总包含整个子树。", "Search hides rows; totals include the whole subtree."))
                Spacer()
                Button(language.text("查找", "Find")) { searchFocused = true }.keyboardShortcut("f", modifiers: .command)
            }.font(LedgerFonts.font(.caption)).foregroundColor(.secondary).padding(.horizontal).padding(.vertical, 6)
            Divider()

            // Tree Header
            HStack {
                Text(language.text("账户名称", "Account"))
                    .font(LedgerFonts.font(.caption))
                    .bold()
                    .foregroundColor(.secondary)
                Spacer()
                Text(language.text("直属余额", "Own balance"))
                    .font(LedgerFonts.font(.caption))
                    .bold()
                    .foregroundColor(.secondary)
                    .frame(width: 140, alignment: .trailing)
                Text(language.text("汇总余额", "Subtree total"))
                    .font(LedgerFonts.font(.caption))
                    .bold()
                    .foregroundColor(.secondary)
                    .frame(width: 160, alignment: .trailing)
            }
            .padding(.horizontal, 20)
            .padding(.vertical, 8)
            .background(Color(NSColor.controlBackgroundColor))

            Divider()

            // Tree Content
            if filteredNodes.isEmpty {
                VStack(spacing: 12) {
                    Spacer()
                    Image(systemName: "tray")
                        .font(.system(size: 36))
                        .foregroundColor(.secondary)
                    Text(language.text("无匹配账户数据", "No matching accounts"))
                        .foregroundColor(.secondary)
                    Spacer()
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                List(selection: $treeState.selectedAccount) {
                    ForEach(visibleNodes) { node in
                        AccountTreeRow(node: node, treeState: treeState, openAccount: openAccount).tag(node.full_name)
                    }
                }
                .listStyle(.inset)
                .focusable()
                .focused($listFocused)
                .simultaneousGesture(TapGesture().onEnded { listFocused = true })
                .onMoveCommand { direction in
                    let rows = visibleNodes
                    guard let name = treeState.selectedAccount,
                          let index = rows.firstIndex(where: { $0.full_name == name }) else {
                        treeState.selectedAccount = rows.first?.full_name
                        return
                    }
                    let node = rows[index]
                    switch direction {
                    case .up:
                        treeState.selectedAccount = rows[max(0, index - 1)].full_name
                    case .down:
                        treeState.selectedAccount = rows[min(rows.count - 1, index + 1)].full_name
                    case .right:
                        if treeState.isExpanded(name) {
                            if let child = node.children.first { treeState.selectedAccount = child.full_name }
                        } else { treeState.collapsedNodes.remove(name) }
                    case .left:
                        if treeState.isExpanded(name) && !node.children.isEmpty && treeState.searchText.isEmpty {
                            treeState.collapsedNodes.insert(name)
                        } else if let colon = name.lastIndex(of: ":") {
                            treeState.selectedAccount = String(name[..<colon])
                        }
                    default: break
                    }
                }
            }
        }
    }

    private func allNames(_ nodes: [AccountTreeNode]) -> [String] { nodes.flatMap { [$0.full_name] + allNames($0.children) } }

    private func filterNode(_ node: AccountTreeNode, query: String) -> AccountTreeNode? {
        let nameMatch = node.full_name.lowercased().contains(query)
        let filteredChildren = node.children.compactMap { filterNode($0, query: query) }
        if nameMatch || !filteredChildren.isEmpty {
            return AccountTreeNode(
                name: node.name,
                fullName: node.full_name,
                depth: node.depth,
                isLeaf: filteredChildren.isEmpty && node.is_leaf,
                explicit: node.explicit,
                directBalances: node.direct_balances,
                subtreeBalances: node.subtree_balances,
                directAmounts: node.direct_amounts, subtreeAmounts: node.subtree_amounts,
                children: filteredChildren, directValuation: node.direct_valuation, subtreeValuation: node.subtree_valuation
            )
        }
        return nil
    }
}

struct AccountTreeRow: View {
    @Environment(\.ledgerLanguage) private var language
    @ObservedObject var node: AccountTreeNode
    @ObservedObject var treeState: AccountTreeState
    var openAccount: (String) -> Void = { _ in }

    var body: some View {
        let isExpanded = treeState.isExpanded(node.full_name)

        VStack(spacing: 0) {
            HStack(spacing: 6) {
                // Indentation
                if node.depth > 0 {
                    Spacer()
                        .frame(width: CGFloat(node.depth) * 18)
                }

                // Disclosure Chevron
                if !node.children.isEmpty {
                    Button(action: {
                        withAnimation(.easeInOut(duration: 0.15)) {
                            treeState.toggleExpanded(node.full_name)
                        }
                    }) {
                        Image(systemName: isExpanded ? "chevron.down" : "chevron.right")
                            .font(.system(size: 11, weight: .bold))
                            .foregroundColor(.secondary)
                            .frame(width: 16, height: 16)
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel(node.full_name)
                    .accessibilityValue(isExpanded ? language.text("已展开", "Expanded") : language.text("已折叠", "Collapsed"))
                } else {
                    Spacer().frame(width: 16)
                }

                // Folder/Leaf Icon
                Image(systemName: node.children.isEmpty ? "doc.text" : "folder.fill")
                    .foregroundColor(node.children.isEmpty ? .secondary : .accentColor)
                    .font(.system(size: 13))

                // Account Name
                Button { openAccount(node.full_name) } label: {
                    Text(node.name).font(LedgerFonts.font(.body)).bold(!node.children.isEmpty)
                }.buttonStyle(.plain)
                    .help(node.full_name)
                    .accessibilityLabel(language.text("账户详情：", "Account details: ") + node.full_name)

                Spacer()

                // Direct Balances
                BalanceColumnView(balances: node.direct_amounts, valuation: node.direct_valuation)
                    .frame(width: 140, alignment: .trailing)

                // Subtree Balances
                BalanceColumnView(balances: node.subtree_amounts, valuation: node.subtree_valuation)
                    .frame(width: 160, alignment: .trailing)
            }
            .padding(.vertical, 4)
            .contentShape(Rectangle())

        }
    }
}

struct BalanceColumnView: View {
    @Environment(\.ledgerLanguage) private var language
    let balances: [String: DisplayAmount]
    var valuation: AccountValuation? = nil

    var body: some View {
        VStack(alignment: .trailing, spacing: 2) {
            if balances.isEmpty {
                Text(valuation?.completeness == "complete" ? "0" : "—")
                    .font(LedgerFonts.font(.caption, design: .monospaced))
                    .foregroundColor(.secondary.opacity(0.5))
            } else {
                ForEach(Array(balances.keys.sorted()), id: \.self) { cur in
                    let amt = balances[cur]!.label
                    let isNegative = amt.hasPrefix("-")
                    HStack(spacing: 4) {
                        Text(amt)
                            .help(balances[cur]!.exact)
                            .font(LedgerFonts.font(.body, design: .monospaced))
                            .bold()
                            .foregroundColor(isNegative ? .red : .primary)
                        Text(cur)
                            .font(LedgerFonts.font(.caption, design: .monospaced))
                            .foregroundColor(.secondary)
                    }
                }
            }
            if let valuation, valuation.mode == "converted", valuation.completeness != "empty" {
                VStack(alignment: .trailing, spacing: 2) {
                    if let total = valuation.complete_total {
                        Text(language.text("折算总计", "Converted total") + ": " + total.label + " " + valuation.target_currency)
                            .help(total.exact).foregroundColor(.secondary)
                    } else if let portion = valuation.valued_portion {
                        Text(language.text("已折算部分", "Valued portion") + ": " + portion.label + " " + valuation.target_currency)
                            .help(portion.exact).foregroundColor(.orange)
                    } else {
                        Text(language.text("折算不可用", "Conversion unavailable")).foregroundColor(.orange)
                    }
                    if !valuation.missing.isEmpty {
                        Text(language.text("缺报价：", "Missing quotes: ") + valuation.missing.map { $0.commodity }.joined(separator: ", "))
                            .foregroundColor(.orange)
                    }
                }.font(LedgerFonts.font(.caption)).fixedSize(horizontal: false, vertical: true)
                    .help(language.text("截至日：", "Through: ") + valuation.as_of + "\n" + valuation.quotes.map { $0.base + "/" + $0.quote + " " + $0.rate.exact + " @ " + $0.date }.joined(separator: "\n"))
            }
        }
    }
}
