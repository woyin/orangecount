import SwiftUI

final class AccountTreeState: ObservableObject {
    @Published var searchText: String = ""
    @Published var collapsedNodes: Set<String> = []

    func isExpanded(_ fullName: String) -> Bool {
        return !collapsedNodes.contains(fullName)
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
    let rootNodes: [AccountTreeNode]
    let operatingCurrencies: [String]
    @Binding var reportType: String
    @StateObject private var treeState = AccountTreeState()

    var filteredNodes: [AccountTreeNode] {
        if treeState.searchText.isEmpty {
            return rootNodes
        }
        return rootNodes.compactMap { filterNode($0, query: treeState.searchText.lowercased()) }
    }

    var body: some View {
        VStack(spacing: 0) {
            // Header Bar
            HStack {
                Picker("报表类型", selection: $reportType) {
                    Text("资产负债表 (Balance Sheet)").tag("balance_sheet")
                    Text("损益表 (Income Statement)").tag("income_statement")
                }
                .pickerStyle(.segmented)
                .frame(maxWidth: 320)

                Spacer()

                HStack {
                    Image(systemName: "magnifyingglass")
                        .foregroundColor(.secondary)
                    TextField("快速筛选账户...", text: $treeState.searchText)
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

            Divider()

            // Tree Header
            HStack {
                Text("账户名称")
                    .font(.caption)
                    .bold()
                    .foregroundColor(.secondary)
                Spacer()
                Text("直属余额")
                    .font(.caption)
                    .bold()
                    .foregroundColor(.secondary)
                    .frame(width: 140, alignment: .trailing)
                Text("汇总余额 (含子账户)")
                    .font(.caption)
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
                    Text("无匹配账户数据")
                        .foregroundColor(.secondary)
                    Spacer()
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                List {
                    ForEach(filteredNodes) { root in
                        AccountTreeRow(node: root, treeState: treeState)
                    }
                }
                .listStyle(.inset)
            }
        }
    }

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
                children: filteredChildren
            )
        }
        return nil
    }
}

struct AccountTreeRow: View {
    @ObservedObject var node: AccountTreeNode
    @ObservedObject var treeState: AccountTreeState

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
                } else {
                    Spacer().frame(width: 16)
                }

                // Folder/Leaf Icon
                Image(systemName: node.children.isEmpty ? "doc.text" : "folder.fill")
                    .foregroundColor(node.children.isEmpty ? .secondary : .accentColor)
                    .font(.system(size: 13))

                // Account Name
                Text(node.name)
                    .font(.system(.body, design: .default))
                    .bold(!node.children.isEmpty)

                Spacer()

                // Direct Balances
                BalanceColumnView(balances: node.direct_balances)
                    .frame(width: 140, alignment: .trailing)

                // Subtree Balances
                BalanceColumnView(balances: node.subtree_balances)
                    .frame(width: 160, alignment: .trailing)
            }
            .padding(.vertical, 4)
            .contentShape(Rectangle())

            // Children recursion
            if isExpanded && !node.children.isEmpty {
                ForEach(node.children) { child in
                    AccountTreeRow(node: child, treeState: treeState)
                }
            }
        }
    }
}

struct BalanceColumnView: View {
    let balances: [String: String]

    var body: some View {
        VStack(alignment: .trailing, spacing: 2) {
            if balances.isEmpty {
                Text("-")
                    .font(.system(.caption, design: .monospaced))
                    .foregroundColor(.secondary.opacity(0.5))
            } else {
                ForEach(Array(balances.keys.sorted()), id: \.self) { cur in
                    let amt = balances[cur] ?? "0"
                    let isNegative = amt.hasPrefix("-")
                    HStack(spacing: 4) {
                        Text(amt)
                            .font(.system(.body, design: .monospaced))
                            .bold()
                            .foregroundColor(isNegative ? .red : .primary)
                        Text(cur)
                            .font(.system(.caption, design: .monospaced))
                            .foregroundColor(.secondary)
                    }
                }
            }
        }
    }
}
