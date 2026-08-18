// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// This dependency-free bundle is the checked-in output of web/src/app.ts.
// It intentionally uses only the versioned local API; no runtime network or
// CDN dependency is needed by the embedded interface.
const translations = {
  en: {
    subtitle: "Read-only local ledger view.", language: "Language", overview: "Overview",
    accounts: "Accounts", journal: "Journal", trialBalance: "Trial balance", balanceSheet: "Balance sheet",
    incomeStatement: "Income statement", holdings: "Holdings", prices: "Prices", commodities: "Commodities", events: "Events",
    documents: "Documents", statistics: "Statistics", diagnostics: "Diagnostics", query: "Query", status: "Status",
    pivot: "Pivot", pivotRows: "Rows", pivotColumns: "Columns", pivotValues: "Values", pivotMonth: "Month",
    pivotQuarter: "Quarter", pivotYear: "Year", pivotNone: "Per currency", pivotRoot1: "Level-1 accounts",
    pivotRoot2: "Level-2 accounts", pivotRoot3: "Level-3 accounts", pivotSum: "Period totals",
    pivotBalance: "Ending balance", pivotAccount: "Account prefix", pivotApply: "Apply",
    pivotFlat: "Flat table", pivotCross: "Cross-tab", pivotNeedsShape: "Pivot view needs one dimension column plus a value column (or two dimensions plus a value).",
    windowLabel: "Window", orderNewest: "Newest first", orderOldest: "Oldest first",
    windowAll: "All entries", windowWeek: "Last week", windowMonth: "Last month",
    windowQuarter: "Last 3 months", windowYear: "Last year", windowCustom: "Custom dates",
    chartWindow: "Time range", chartWindow3m: "3M", chartWindow6m: "6M", chartWindow1y: "1Y",
    chartWindow3y: "3Y", chartWindowAll: "All",
    snapshot: "Snapshot", valid: "Valid", accountsCount: "Accounts", diagnosticsCount: "Diagnostics",
    publishedAt: "Published", yes: "yes", no: "no", loading: "Loading…", unavailable: "No valid snapshot.",
    rows: "rows", columns: "columns", run: "Run query", queryHint: "SELECT account, balance FROM accounts ORDER BY account",
    empty: "No rows.", requestFailed: "Request failed", source: "Source", file: "File", content: "Content",
    sourceHint: "Browse files in the resolved include graph.",
    documentsHint: "Document attachments are served only from configured roots.",
    from: "From", to: "To", apply: "Apply", reset: "Reset", approximate: "approximate", exact: "Exact value",
    period: "Period", valuation: "Valuation", allPeriods: "All periods", monthly: "Monthly", quarterly: "Quarterly", yearly: "Yearly",
    atCost: "At cost", marketValue: "Market value", exportCSV: "Export CSV", chart: "Chart", tree: "Account tree",
    flag: "Flag", tag: "Tag", link: "Link", payee: "Payee", narration: "Narration", expand: "Expand details",
    save: "Save", saved: "Saved queries", queryName: "Query name", download: "Download", previewDiagnostics: "Preview diagnostics; commit will revalidate", unavailablePrice: "Unavailable: no local price", unavailableCurrency: "Unavailable: no conversion quote", survivingLots: "As-of uses surviving lots", previousPage: "Previous page", nextPage: "Next page", pageOf: "of",
    editor: "Editor", import: "Import", options: "Options", help: "Help", files: "Files", validate: "Validate", time: "Time", currency: "Currency", adapter: "Adapter", offset: "Offset",
    commit: "Commit", preview: "Preview", target: "Target ledger", chooseFile: "Choose a local file", backup: "Backup",
    discard: "Discard", syntax: "Syntax preview", noFile: "Select a file", searchHelp: "Search help", back: "Back",
    summary: "Summary", leavesOnly: "Leaves only", hierarchy: "Hierarchy", treemap: "Treemap", sunburst: "Sunburst", icicle: "Icicle",
    accountBalance: "Account balance", subtreeTotal: "Subtree total", notValued: "Not valued",
    transaction: "Transaction", note: "Note", document: "Document", open: "Open", close: "Close",
    pad: "Pad", price: "Price", queryKind: "Query", custom: "Custom", commodity: "Commodity",
    event: "Event", metadata: "Metadata", postings: "Postings", details: "Details",
    runningBalance: "Running balance", theme: "Theme", system: "System", dark: "Dark", light: "Light",
    optionsFromLedger: "Ledger options", extended: "Extended", accountDetail: "Account detail",
    noConversionData: "No data for this currency", operating: "Operating",
    fixFirst: "Fix first", recheckAfter: "Recheck after source fixes", repairOrderHint: "Fix source and syntax problems first, then validate again before reviewing accounting diagnostics.",
    learnHowToFix: "Learn how to fix", hideGuidance: "Hide guidance", loadingGuide: "Loading repair guidance…", noGuidance: "Repair guidance is unavailable for this code.",
    whatHappened: "What happened", whyBlocks: "Why it blocks", whereToInspect: "Where to inspect", safeChecks: "Safe checks and changes", genericExample: "Generic example", before: "Before", after: "After", nextStep: "Next step", helpTopic: "Open full help topic", showLocalContext: "Show local context", contextUnavailable: "Local context is unavailable.",
  },
  "zh-CN": {
    subtitle: "只读本地账本视图。", language: "语言", overview: "概览", accounts: "账户", journal: "日记账",
    trialBalance: "试算平衡", balanceSheet: "资产负债表", incomeStatement: "损益表", holdings: "持仓",
    prices: "价格", commodities: "商品", events: "事件", documents: "文档", statistics: "统计", diagnostics: "诊断", query: "查询", status: "状态",
    pivot: "透视表", pivotRows: "行", pivotColumns: "列", pivotValues: "值", pivotMonth: "按月",
    pivotQuarter: "按季", pivotYear: "按年", pivotNone: "按币种", pivotRoot1: "一级科目",
    pivotRoot2: "二级科目", pivotRoot3: "三级科目", pivotSum: "期间合计", pivotBalance: "期末余额",
    pivotAccount: "账户前缀", pivotApply: "应用", pivotFlat: "平铺表", pivotCross: "交叉表",
    pivotNeedsShape: "透视视图需要一列维度加一列数值（或两列维度加一列数值）。",
    windowLabel: "范围", orderNewest: "最新在前", orderOldest: "最旧在前",
    windowAll: "全部记录", windowWeek: "最近一周", windowMonth: "最近一月",
    windowQuarter: "最近三月", windowYear: "最近一年", windowCustom: "自定义日期",
    chartWindow: "时间轴", chartWindow3m: "近3月", chartWindow6m: "近6月", chartWindow1y: "近1年",
    chartWindow3y: "近3年", chartWindowAll: "全部",
    snapshot: "快照", valid: "有效", accountsCount: "账户数", diagnosticsCount: "诊断数", publishedAt: "发布时间",
    yes: "是", no: "否", loading: "加载中…", unavailable: "没有有效快照。", rows: "行", columns: "列",
    run: "运行查询", queryHint: "SELECT account, balance FROM accounts ORDER BY account", empty: "没有数据。",
    requestFailed: "请求失败", source: "源文件", file: "文件", content: "内容",
    sourceHint: "浏览已解析 include 图中的文件。", documentsHint: "文档附件仅从已配置的根目录提供。",
    from: "起始日期", to: "结束日期", apply: "应用", reset: "重置", approximate: "近似值", exact: "精确值",
    period: "期间", valuation: "估值", allPeriods: "全部期间", monthly: "按月", quarterly: "按季度", yearly: "按年",
    atCost: "按成本", marketValue: "按市值", exportCSV: "导出 CSV", chart: "图表", tree: "账户树",
    flag: "标记", tag: "标签", link: "链接", payee: "收款方", narration: "摘要", expand: "展开详情",
    save: "保存", saved: "已保存查询", queryName: "查询名称", download: "下载", previewDiagnostics: "预览存在诊断；提交时会重新验证", unavailablePrice: "不可用：没有本地价格", unavailableCurrency: "不可用：没有换算报价", survivingLots: "截至日期使用当前剩余批次", previousPage: "上一页", nextPage: "下一页", pageOf: "/",
    editor: "编辑器", import: "导入", options: "选项", help: "帮助", files: "文件", validate: "验证", time: "时间", currency: "货币", adapter: "适配器", offset: "抵销账户",
    commit: "提交", preview: "预览", target: "目标账本", chooseFile: "选择本地文件", backup: "备份",
    discard: "丢弃", syntax: "语法预览", noFile: "选择文件", searchHelp: "搜索帮助", back: "返回",
    summary: "汇总", leavesOnly: "仅叶子账户", hierarchy: "层级", treemap: "矩形树图", sunburst: "旭日图", icicle: "冰柱图",
    accountBalance: "账户余额", subtreeTotal: "含子账户汇总", notValued: "未估值",
    transaction: "交易", note: "备注", document: "文档", open: "开户", close: "销户",
    pad: "补差", price: "价格", queryKind: "查询", custom: "自定义", commodity: "商品",
    event: "事件", metadata: "元数据", postings: "过账", details: "详情",
    runningBalance: "运行结存", theme: "主题", system: "跟随系统", dark: "深色", light: "浅色",
    optionsFromLedger: "账本选项", extended: "扩展", accountDetail: "账户详情",
    noConversionData: "该货币暂无数据", operating: "记账本位币",
    fixFirst: "先处理", recheckAfter: "修复源文件后重新检查", repairOrderHint: "请先处理源文件和语法问题，再重新验证，然后检查会计语义诊断。",
    learnHowToFix: "查看修复指导", hideGuidance: "隐藏指导", loadingGuide: "正在加载修复指导…", noGuidance: "此错误码暂无修复指导。",
    whatHappened: "发生了什么", whyBlocks: "为什么阻塞", whereToInspect: "去哪里检查", safeChecks: "安全检查与修改", genericExample: "通用示例", before: "修改前", after: "修改后", nextStep: "下一步", helpTopic: "打开完整帮助主题", showLocalContext: "显示本地上下文", contextUnavailable: "本地上下文不可用。",
  },
};

const reportRoutes = [
  ["accounts", "accounts"], ["journal", "journal"], ["trial-balance", "trialBalance"],
  ["balance-sheet", "balanceSheet"], ["income-statement", "incomeStatement"], ["holdings", "holdings"],
  ["commodities", "commodities"], ["prices", "prices"], ["events", "events"], ["documents", "documents"],
  ["statistics", "statistics"],
];
const navRoutes = [["overview", "overview"], ...reportRoutes, ["source", "source"], ["diagnostics", "diagnostics"], ["query", "query"], ["pivot", "pivot"], ["editor", "editor"], ["import", "import"], ["options", "options"], ["help", "help"]];
const params = new URLSearchParams(window.location.search);
let locale = params.get("locale") === "zh-CN" ? "zh-CN" : (localStorage.getItem("orangecount-locale") || "en");
if (!translations[locale]) locale = "en";
const pathViews = {
  "/": "overview",
  "/accounts": "accounts",
  "/journal": "journal",
  "/trial_balance": "trial-balance",
  "/trial-balance": "trial-balance",
  "/balance_sheet": "balance-sheet",
  "/balance-sheet": "balance-sheet",
  "/income_statement": "income-statement",
  "/income-statement": "income-statement",
  "/holdings": "holdings",
  "/account": "accounts",
  "/commodities": "commodities",
  "/prices": "prices",
  "/events": "events",
  "/documents": "documents",
  "/statistics": "statistics",
  "/query": "query",
  "/diagnostics": "diagnostics",
  "/source": "source",
  "/editor": "editor",
  "/import": "import",
  "/options": "options",
  "/help": "help",
};
let helpPage = "";
if (window.location.pathname.startsWith("/help/")) {
  try { helpPage = decodeURIComponent(window.location.pathname.slice("/help/".length)); } catch (_) { helpPage = window.location.pathname.slice("/help/".length); }
}
const accountPath = window.location.pathname.match(/^\/account\/(.+)$/);
let accountFromPath = "";
if (accountPath) {
  try { accountFromPath = decodeURIComponent(accountPath[1]); } catch (_) { accountFromPath = accountPath[1]; }
}
let view = helpPage ? "help" : (params.get("view") || (accountPath ? "accounts" : pathViews[window.location.pathname]) || "overview");
if (!navRoutes.some(([route]) => route === view)) view = "overview";
let globalState = {
  time: params.get("time") || "all",
  account: params.get("account") || accountFromPath,
  filter: params.get("filter") || "",
  currency: params.get("currency") || "",
};
let reportState = {
  period: params.get("period") || "all",
  valuation: params.get("valuation") || "at-cost",
  asOf: params.get("as_of") || "",
  treeMode: params.get("tree_mode") === "leaves" ? "leaves" : "hierarchy",
  hierarchyLayout: params.get("hierarchy_layout") || "treemap",
};
let journalRange = { from: params.get("from") || "", to: params.get("to") || "" };
// Fava's journal defaults to newest-first with 1000-entry pages; we keep the
// newest-first default and slice a recent window client-side (FD-0007).
let journalOrder = params.get("order") === "asc" ? "asc" : "desc";
let journalWindow = journalRange.from || journalRange.to ? "custom" : (params.get("window") || "month");
let journalAnchorDate = "";
// Statement charts default to the last year of points; Fava draws the full
// ledger span (FD-0008).
let chartWindowState = params.get("chart_window") || "1y";
let journalFilters = {
  flag: params.get("flag") || "",
  tag: params.get("tag") || "",
  link: params.get("link") || "",
  payee: params.get("payee") || "",
  narration: params.get("narration") || "",
  kind: params.get("kind") || "",
};
const journalPageParam = Number.parseInt(params.get("page") || "1", 10);
let journalTableState = {
  sort: params.get("sort") || "",
  direction: params.get("direction") === "descending" ? "descending" : (params.get("sort") ? "ascending" : ""),
  page: Number.isFinite(journalPageParam) && journalPageParam > 0 ? journalPageParam - 1 : 0,
};

const app = document.getElementById("app");
const navigation = document.getElementById("navigation");
const localePicker = document.getElementById("locale");
const subtitle = document.getElementById("subtitle");
const pageTitle = document.getElementById("page-title");
const brand = document.getElementById("brand");
let ledgerTitle = "";
let ledgerOptions = {};
let operatingCurrency = "";
let diagnosticCount = 0;
let theme = localStorage.getItem("orangecount-theme") || "dark";
const sidebar = document.getElementById("sidebar");
const menuToggle = document.getElementById("menu-toggle");
const timePicker = document.getElementById("global-time");
const accountPicker = document.getElementById("global-account");
const globalFilter = document.getElementById("global-filter");
const currencySwitch = document.getElementById("currency-switch");

function t(key) { return (translations[locale] && translations[locale][key]) || translations.en[key] || key; }
function escapeHTML(value) {
  return String(value == null ? "" : value).replace(/[&<>"']/g, (character) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[character]));
}
function presented(value) {
  if (value && typeof value === "object" && typeof value.display === "string" && typeof value.exact === "string") return value;
  return null;
}
function display(value) {
  const decimal = presented(value);
  if (decimal) return decimal.display;
  if (Array.isArray(value)) return value.join(", ");
  if (value && typeof value === "object") return JSON.stringify(value);
  return value == null ? "" : String(value);
}
function exactValue(value) {
  const decimal = presented(value);
  return decimal ? decimal.exact : display(value);
}
function decimalParts(value) {
  const text = String(value).trim();
  if (/^[+-]?\d+$/.test(text)) return { numerator: BigInt(text), denominator: 1n };
  const fraction = text.match(/^([+-]?\d+)\/([+-]?\d+)$/);
  if (fraction && fraction[2] !== "0") {
    const numerator = BigInt(fraction[1]);
    const denominator = BigInt(fraction[2]);
    return denominator < 0n ? { numerator: -numerator, denominator: -denominator } : { numerator, denominator };
  }
  const decimal = text.match(/^([+-]?)(\d+)(?:\.(\d+))?$/);
  if (!decimal) return null;
  const digits = decimal[3] || "";
  const denominator = 10n ** BigInt(digits.length);
  const sign = decimal[1] === "-" ? -1n : 1n;
  const numerator = sign * BigInt(`${decimal[2]}${digits}`);
  return { numerator, denominator };
}
function compareTyped(left, right, kind) {
  if (left === right) return 0;
  if (kind === "date") return left < right ? -1 : 1;
  if (kind === "decimal") {
    const a = decimalParts(left);
    const b = decimalParts(right);
    if (a && b) {
      const comparison = a.numerator * b.denominator - b.numerator * a.denominator;
      if (comparison !== 0n) return comparison < 0n ? -1 : 1;
      return 0;
    }
  }
  return left < right ? -1 : 1;
}
function cellKind(value, column) {
  const exact = exactValue(value);
  if (/^\d{4}-\d{2}-\d{2}$/.test(exact) || column === "date") return "date";
  if (presented(value) || decimalParts(exact)) return "decimal";
  return "text";
}
function renderCell(value, column) {
  const decimal = presented(value);
  const exact = exactValue(value);
  const kind = cellKind(value, column);
  const title = decimal ? ` title="${escapeHTML(`${t("exact")}: ${decimal.exact}`)}"` : "";
  const marker = decimal && decimal.approximate ? `<span aria-label="${escapeHTML(t("approximate"))}">≈</span> ` : "";
  if (column === "valuation_status" && typeof value === "string") {
    const label = value === "unavailable-price" ? t("unavailablePrice") : value === "unavailable-currency" ? t("unavailableCurrency") : value;
    return `<td data-sort-kind="text" data-sort-value="${escapeHTML(value)}" class="${value.startsWith("unavailable") ? "warning" : ""}">${escapeHTML(label)}</td>`;
  }
  if (column === "filename" && value) return `<td data-sort-kind="${kind}" data-sort-value="${escapeHTML(exact)}"${title}><a href="/documents/${encodeURIComponent(value)}">${escapeHTML(value)}</a></td>`;
  if ((column === "file" || column === "path") && typeof value === "string" && value) return `<td data-sort-kind="${kind}" data-sort-value="${escapeHTML(exact)}"${title}><a href="/?view=source&path=${encodeURIComponent(value)}">${escapeHTML(value)}</a></td>`;
  if (column === "account" && typeof value === "string" && value) {
    const target = new URL(window.location.href);
    target.searchParams.set("view", "accounts");
    target.searchParams.set("account", value);
    return `<td data-sort-kind="${kind}" data-sort-value="${escapeHTML(exact)}"${title}><a href="${escapeHTML(`${target.pathname}?${target.searchParams.toString()}`)}">${escapeHTML(value)}</a></td>`;
  }
  if ((column === "tags" || column === "links") && Array.isArray(value) && value.length) {
    const details = value.map((item) => `<code>${escapeHTML(item)}</code>`).join(", ");
    return `<td data-sort-kind="${kind}" data-sort-value="${escapeHTML(exact)}"${title}><details><summary>${value.length}</summary>${details}</details></td>`;
  }
  return `<td data-sort-kind="${kind}" data-sort-value="${escapeHTML(exact)}"${title}>${marker}${escapeHTML(display(value))}</td>`;
}
function treeMetadata(rows) {
  // Account names are labels, not a tree by themselves. Only accept a
  // declared parent when that parent is represented by another row in this
  // result. This keeps indentation/collapse state tied to report-provided
  // relationships and avoids treating a colon in an unrelated field as
  // hierarchy syntax.
  const accounts = new Set(rows.map((row) => typeof row.account === "string" ? row.account : "").filter(Boolean));
  const parentByAccount = new Map();
  rows.forEach((row) => {
    const account = typeof row.account === "string" ? row.account : "";
    const candidate = typeof row._tree_parent === "string" ? row._tree_parent : "";
    if (account && candidate && candidate !== account && accounts.has(candidate)) parentByAccount.set(account, candidate);
  });
  const children = new Set();
  parentByAccount.forEach((parent) => children.add(parent));
  const depthFor = (account) => {
    let depth = 0;
    const seen = new Set([account]);
    let parent = parentByAccount.get(account) || "";
    while (parent && !seen.has(parent)) {
      depth += 1;
      seen.add(parent);
      parent = parentByAccount.get(parent) || "";
    }
    return depth;
  };
  return new Map(rows.map((row) => {
    const account = typeof row.account === "string" ? row.account : "";
    return [row, { account, parent: parentByAccount.get(account) || "", depth: account ? depthFor(account) : 0, hasChild: children.has(account) }];
  }));
}
// pivotCurrency groups a per-(account, currency) report into one row per
// account with a dynamic per-currency column, matching Fava's per-column
// currency presentation. The operating currency is listed first when present,
// then currencies in ledger appearance order. Accounts not holding a currency
// get an empty cell instead of removing the row. Aggregate rows expose their
// subtree total; leaf rows expose their own balance.
function pivotCurrency(rows, columns, preferredCurrency) {
  const currencyOrder = [];
  const seen = new Set();
  if (preferredCurrency && !seen.has(preferredCurrency)) {
    seen.add(preferredCurrency);
    currencyOrder.push(preferredCurrency);
  }
  rows.forEach((row) => {
    const currency = row.currency;
    if (currency && !seen.has(currency)) {
      seen.add(currency);
      currencyOrder.push(currency);
    }
  });
  const accountOrder = [];
  const byAccount = new Map();
  rows.forEach((row) => {
    const account = row.account;
    if (!byAccount.has(account)) {
      byAccount.set(account, []);
      accountOrder.push(account);
    }
    byAccount.get(account).push(row);
  });
  const baseColumns = columns.filter((column) => column !== "currency" && column !== "balance" && column !== "own_balance" && column !== "total_balance");
  const newColumns = [...baseColumns, ...currencyOrder];
  const outRows = accountOrder.map((account) => {
    const group = byAccount.get(account);
    const first = group[0];
    const row = {};
    baseColumns.forEach((column) => { row[column] = first[column]; });
    Object.keys(first).forEach((key) => {
      if (key.startsWith("_tree_")) row[key] = first[key];
    });
    row.account = account;
    const role = first._tree_role;
    currencyOrder.forEach((currency) => {
      const found = group.find((item) => item.currency === currency);
      if (!found) {
        row[currency] = null;
        return;
      }
      let value = found.balance;
      if (role === "aggregate") {
        if (found.total_balance != null) value = found.total_balance;
      } else if (found.own_balance != null) {
        value = found.own_balance;
      }
      row[currency] = value;
    });
    return row;
  });
  return { rows: outRows, columns: newColumns };
}
function renderTable(result, options = {}) {
  if (!result || !Array.isArray(result.columns)) return `<p class="muted">${escapeHTML(t("empty"))}</p>`;
  // Tree metadata is intentionally part of the API payload but not of the
  // visible accounting table.
  let columns = result.columns.filter((column) => !column.startsWith("_tree_"));
  let rows = Array.isArray(result.rows) ? result.rows : [];
  if (!rows.length) return `<p class="muted">${escapeHTML(t("empty"))}</p>`;
  const tree = options.tree === true;
  // Currency pivot: for account tree reports, group rows by account and expand
  // the currency column into dynamic per-currency columns (Fava-style). The
  // global currency switch is a display preference and must not gate which
  // currency columns are visible.
  if (options.pivotCurrency === true && columns.includes("account") && columns.includes("currency")) {
    const pivoted = pivotCurrency(rows, columns, operatingCurrency);
    rows = pivoted.rows;
    columns = pivoted.columns;
  }
  const hasSplit = tree && columns.includes("own_balance") && columns.includes("total_balance");
  if (hasSplit) {
    columns = columns.filter((column) => column !== "balance");
  }
  const metadata = tree ? treeMetadata(rows) : new Map();
  const leavesOnly = tree && options.leavesOnly === true;
  const visibleRows = leavesOnly ? rows.filter((row) => row._tree_role === "direct") : rows;
  return `<p class="muted">${visibleRows.length} ${escapeHTML(t("rows"))} · ${columns.length} ${escapeHTML(t("columns"))}</p><div class="table-wrap"><table data-sortable="true"${tree ? " data-tree-table=\"true\"" : ""}><thead><tr>${columns.map((column, index) => `<th scope="col"><button type="button" class="table-sort" data-column-index="${index}" data-column-name="${escapeHTML(column)}" aria-sort="none">${escapeHTML(column)}</button></th>`).join("")}</tr></thead><tbody>${visibleRows.map((row) => {
    const account = typeof row.account === "string" ? row.account : "";
    const treeRow = tree ? metadata.get(row) : null;
    const depth = treeRow ? treeRow.depth : 0;
    const parent = treeRow ? treeRow.parent : "";
    const hasChild = !!treeRow && treeRow.hasChild;
    const role = tree && typeof row._tree_role === "string" ? row._tree_role : "";
    return `<tr${tree ? ` data-tree-depth="${depth}" data-tree-account="${escapeHTML(account)}" data-tree-parent="${escapeHTML(parent)}" data-tree-role="${escapeHTML(role)}"` : ""}>${columns.map((column) => {
      let cellValue = row[column];
      // On an aggregate row the "own balance" column is the subtree total, so
      // the parent is clearly a summary rather than a duplicate ordinary row.
      if (hasSplit && column === "own_balance" && role === "aggregate") cellValue = row.total_balance;
      let cell = renderCell(cellValue, column);
      if (tree && column === "account" && account) {
        const summary = hasSplit && role === "aggregate" ? `<span class="tree-aggregate-badge">${escapeHTML(t("summary"))}</span>` : "";
        const toggle = hasChild ? `<button type="button" class="tree-toggle" data-tree-toggle="${escapeHTML(account)}" aria-expanded="true" aria-label="${escapeHTML(`${t("tree")}: ${account}`)}">▾</button>` : `<span class="tree-toggle-placeholder" aria-hidden="true"></span>`;
        cell = cell.replace(">", `><span class="tree-indent" aria-hidden="true" style="--tree-depth:${depth}"></span>${toggle}${summary}`);
      }
      return cell;
    }).join("")}</tr>`;
  }).join("")}</tbody></table></div>`;
}
function tableSortKind(table, columnIndex) {
  const cells = Array.from(table.tBodies[0].rows).map((row) => row.cells[columnIndex]).filter(Boolean);
  const cell = cells.find((candidate) => candidate.dataset.sortValue !== "") || cells[0];
  return cell?.dataset.sortKind || "text";
}
function sortTableRows(table, columnIndex, direction) {
  const kind = tableSortKind(table, columnIndex);
  const rows = Array.from(table.tBodies[0].rows).map((row, index) => ({ row, index }));
  rows.sort((left, right) => {
    const a = left.row.cells[columnIndex];
    const b = right.row.cells[columnIndex];
    const comparison = compareTyped(a?.dataset.sortValue || a?.textContent || "", b?.dataset.sortValue || b?.textContent || "", kind);
    return comparison === 0 ? left.index - right.index : (direction === "ascending" ? comparison : -comparison);
  });
  rows.forEach(({ row }) => table.tBodies[0].appendChild(row));
}
function wireTables(root, options = {}) {
  root.querySelectorAll("table[data-sortable]").forEach((table) => {
    const buttons = Array.from(table.querySelectorAll("button.table-sort"));
    const applySort = (button, direction) => {
      const columnIndex = Number(button.dataset.columnIndex);
      table.querySelectorAll("button.table-sort").forEach((header) => { header.dataset.direction = ""; header.setAttribute("aria-sort", "none"); });
      button.dataset.direction = direction;
      button.setAttribute("aria-sort", direction);
      sortTableRows(table, columnIndex, direction);
      table.dispatchEvent(new CustomEvent("table-sort"));
      if (typeof options.onSort === "function") options.onSort({ column: button.dataset.columnName || String(columnIndex), direction });
    };
    buttons.forEach((button) => button.addEventListener("click", () => {
      const direction = button.dataset.direction === "ascending" ? "descending" : "ascending";
      applySort(button, direction);
    }));
    if (options.sortColumn) {
      const initial = buttons.find((button) => button.dataset.columnName === options.sortColumn || button.dataset.columnIndex === options.sortColumn);
      if (initial) applySort(initial, options.sortDirection === "descending" ? "descending" : "ascending");
    }
  });
}
function wireTree(root) {
  root.querySelectorAll("table[data-tree-table]").forEach((table) => {
    const expanded = new Set();
    table.querySelectorAll("[data-tree-toggle]").forEach((button) => expanded.add(button.dataset.treeToggle));
    const apply = () => {
      table.tBodies[0].querySelectorAll("tr[data-tree-account]").forEach((row) => {
        const parent = row.dataset.treeParent || "";
        let visible = true;
        let cursor = parent;
        while (cursor) {
          if (!expanded.has(cursor)) { visible = false; break; }
          const parentRow = table.tBodies[0].querySelector(`tr[data-tree-account="${CSS.escape(cursor)}"]`);
          cursor = parentRow ? (parentRow.dataset.treeParent || "") : "";
        }
        row.hidden = !visible;
      });
      table.querySelectorAll("[data-tree-toggle]").forEach((button) => button.setAttribute("aria-expanded", String(expanded.has(button.dataset.treeToggle))));
    };
    table.querySelectorAll("[data-tree-toggle]").forEach((button) => button.addEventListener("click", () => {
      const name = button.dataset.treeToggle;
      if (expanded.has(name)) expanded.delete(name); else expanded.add(name);
      apply();
    }));
    apply();
  });
}
function mountTable(container, result, options = {}) {
  container.innerHTML = renderTable(result, options);
  wireTables(container, options);
  wireTree(container);
  if (options.paginate) wirePagination(container, options.pageSize || 50, options);
}
function wireCharts(root) {
  // Hierarchy nodes drill into the account page.
  root.querySelectorAll(".report-hierarchy-chart [data-account]").forEach((node) => {
    node.setAttribute("tabindex", "0");
    const drill = () => {
      const account = node.dataset.account;
      if (!account) return;
      globalState.account = account;
      updateURL();
      renderReport(view);
    };
    node.addEventListener("click", drill);
    node.addEventListener("keydown", (event) => { if (event.key === "Enter" || event.key === " ") { event.preventDefault(); drill(); } });
  });
  // Legend entries toggle their series.
  root.querySelectorAll(".chart-legend [data-series-toggle]").forEach((legendItem) => {
    legendItem.setAttribute("tabindex", "0");
    const index = Number(legendItem.dataset.seriesToggle);
    const toggleSeries = () => {
      // Scope to the legend's own card so stacked chart cards toggle
      // independently.
      const svg = legendItem.closest(".chart-card")?.querySelector(".report-chart") || root.querySelector(".report-chart");
      if (!svg) return;
      const hidden = legendItem.classList.toggle("series-hidden");
      const chart = svg.__chartData;
      const seriesEls = svg.querySelectorAll(`[data-series-index="${index}"]`);
      seriesEls.forEach((el) => { el.classList.toggle("series-hidden", hidden); });
      if (chart) {
        const series = chart.series[index];
        if (series) series.hidden = hidden;
      }
    };
    legendItem.addEventListener("click", toggleSeries);
    legendItem.addEventListener("keydown", (event) => { if (event.key === "Enter" || event.key === " ") { event.preventDefault(); toggleSeries(); } });
  });
  // Points and bars are focusable and expose a tooltip on hover/focus.
  root.querySelectorAll(".report-chart [data-point-value]").forEach((point) => {
    point.setAttribute("tabindex", "0");
    const show = (event) => {
      const svg = point.closest(".report-chart");
      const unit = svg ? svg.dataset.chartUnit || "" : "";
      const seriesIndex = point.dataset.seriesIndex;
      const label = seriesIndex != null ? (point.closest("svg")?.__chartData?.series?.[Number(seriesIndex)]?.label || "") : "";
      const tooltip = buildTooltipHtml(label, point.dataset.pointDate, point.dataset.pointValue, unit, "");
      const box = document.createElement("div");
      box.className = "chart-tooltip";
      box.innerHTML = tooltip;
      box.style.position = "absolute";
      const rect = point.getBoundingClientRect();
      box.style.left = `${rect.left + rect.width / 2}px`;
      box.style.top = `${rect.top}px`;
      document.body.appendChild(box);
      point.__tooltip = box;
    };
    const hide = () => { if (point.__tooltip) { point.__tooltip.remove(); point.__tooltip = null; } };
    point.addEventListener("mouseover", show);
    point.addEventListener("focus", show);
    point.addEventListener("mouseout", hide);
    point.addEventListener("blur", hide);
  });
}
function mountChartData(svg, chart) {
  if (svg) svg.__chartData = chart;
}
function wirePagination(container, pageSize, options = {}) {
  const table = container.querySelector("table");
  if (!table || !table.tBodies[0]) {
    if (typeof options.onPageChange === "function") options.onPageChange(0);
    return;
  }
  let page = Number.isInteger(options.page) && options.page >= 0 ? options.page : 0;
  const rows = () => Array.from(table.tBodies[0].rows);
  if (rows().length <= pageSize) {
    page = 0;
    if (typeof options.onPageChange === "function") options.onPageChange(page);
    return;
  }
  const controls = document.createElement("div");
  controls.className = "table-pagination toolbar";
  const previous = document.createElement("button");
  previous.type = "button";
  previous.textContent = "‹";
  previous.setAttribute("aria-label", t("previousPage"));
  const next = document.createElement("button");
  next.type = "button";
  next.textContent = "›";
  next.setAttribute("aria-label", t("nextPage"));
  const label = document.createElement("span");
  label.className = "muted";
  controls.append(previous, label, next);
  table.parentElement.parentElement.insertBefore(controls, table.parentElement);
  const render = () => {
    const currentRows = rows();
    const pages = Math.max(1, Math.ceil(currentRows.length / pageSize));
    page = Math.max(0, Math.min(page, pages - 1));
    currentRows.forEach((row, index) => { row.hidden = index < page * pageSize || index >= (page + 1) * pageSize; });
    label.textContent = `${page + 1} ${t("pageOf")} ${pages}`;
    previous.disabled = page === 0;
    next.disabled = page === pages - 1;
    if (typeof options.onPageChange === "function") options.onPageChange(page);
  };
  table.addEventListener("table-sort", render);
  previous.addEventListener("click", () => { page -= 1; render(); });
  next.addEventListener("click", () => { page += 1; render(); });
  render();
}
function chartValue(value) {
  const exact = exactValue(value);
  if (!exact) return NaN;
  if (exact.includes("/")) {
    const parts = exact.split("/");
    const numerator = Number(parts[0]);
    const denominator = Number(parts[1]);
    return denominator === 0 ? NaN : numerator / denominator;
  }
  return Number(exact);
}
// Chart helpers are pure functions so the embedded bundle can be regression
// tested from Node without a DOM: they decide ticks, labels, sampling, tooltip
// content, and legend toggle state, and the renderers only place them in SVG.
function ticks(min, max, count = 4) {
  if (!Number.isFinite(min) || !Number.isFinite(max) || min === max) return [min];
  const span = max - min;
  const step = Math.pow(10, Math.floor(Math.log10(span / Math.max(1, count))));
  const niceStep = step * (span / step >= count * 5 ? 5 : span / step >= count * 2 ? 2 : 1);
  const start = Math.ceil(min / niceStep) * niceStep;
  const values = [];
  for (let value = start; value <= max + niceStep * 1e-9; value += niceStep) values.push(value);
  return values;
}
function formatCompact(value) {
  if (!Number.isFinite(value)) return "";
  const abs = Math.abs(value);
  if (abs >= 1e9) return `${(value / 1e9).toFixed(1)}B`;
  if (abs >= 1e6) return `${(value / 1e6).toFixed(1)}M`;
  if (abs >= 1e3) return `${(value / 1e3).toFixed(1)}k`;
  return String(Math.round(value * 100) / 100);
}
function dateTicks(dates, count = 6) {
  const unique = [...new Set(dates)].filter(Boolean).sort();
  if (unique.length <= count) return unique;
  const step = Math.ceil(unique.length / count);
  const result = [];
  for (let index = 0; index < unique.length; index += step) result.push(unique[index]);
  if (result[result.length - 1] !== unique[unique.length - 1]) result.push(unique[unique.length - 1]);
  return result;
}
// samplePoints keeps the first and last points and downsamples the middle so
// precise tooltips survive but dense series do not overflow the viewport.
function samplePoints(points, maxPoints = 60) {
  if (!Array.isArray(points) || points.length <= maxPoints) return points || [];
  const step = (points.length - 1) / (maxPoints - 1);
  const sampled = [];
  for (let index = 0; index < maxPoints; index += 1) {
    sampled.push(points[Math.round(index * step)]);
  }
  return sampled;
}
function buildTooltipHtml(label, date, value, unit, valuation) {
  const parts = [];
  if (label) parts.push(`<strong>${escapeHTML(label)}</strong>`);
  if (date) parts.push(escapeHTML(date));
  parts.push(`<strong>${escapeHTML(String(value))} ${escapeHTML(unit || "")}</strong>`);
  if (valuation) parts.push(`<span class="muted">${escapeHTML(valuation)}</span>`);
  return `<div class="chart-tooltip">${parts.join("<br>")}</div>`;
}
function legendToggleStates(series, hidden) {
  const hiddenSet = hidden instanceof Set ? hidden : new Set(hidden || []);
  return series.map((item, index) => ({ index, label: item.label, hidden: hiddenSet.has(index) }));
}
function chartPeriodLabel() {
  if (reportState.period === "month") return t("monthly");
  if (reportState.period === "quarter") return t("quarterly");
  if (reportState.period === "year") return t("yearly");
  if (globalState.time === "month") return t("monthly");
  if (globalState.time === "year") return t("yearly");
  return t("allPeriods");
}
function semanticChartPoints(result, route) {
  if (!result || !Array.isArray(result.rows)) return { points: [], title: "", unit: "" };
  if (route === "statistics") {
    return { points: result.rows.map((row) => ({ label: String(row.directive || ""), value: chartValue(row.count) })).filter((point) => Number.isFinite(point.value)), title: t("statistics"), unit: t("rows"), period: t("allPeriods") };
  }
  if (route === "balance-sheet" || route === "income-statement") {
    const roots = route === "balance-sheet" ? ["Assets", "Liabilities", "Equity"] : ["Income", "Expenses"];
    const preferred = globalState.currency;
    const selectedExists = result.rows.some((row) => roots.includes(String(row.account || "")) && row.currency === preferred);
    const byRoot = new Map();
    result.rows.forEach((row) => {
      const account = String(row.account || "");
      // Only the explicit top-level row is the composition total. Descendants
      // must not be added again, otherwise a parent balance is double-counted.
      if (!roots.includes(account) || (selectedExists && row.currency !== preferred)) return;
      if (!byRoot.has(account)) byRoot.set(account, { label: account, value: chartValue(row.total_balance != null ? row.total_balance : row.balance), currency: String(row.currency || "") });
    });
    const points = roots.map((root) => byRoot.get(root)).filter((point) => point && Number.isFinite(point.value));
    const unit = points[0] ? points[0].currency : preferred;
    return { points, title: route === "balance-sheet" ? t("balanceSheet") : t("incomeStatement"), unit, period: chartPeriodLabel() };
  }
  return { points: [], title: "", unit: "" };
}
function chartSeriesPoints(chart) {
  if (!chart || !Array.isArray(chart.series)) return [];
  return chart.series.map((series) => ({
    label: String(series.label || ""),
    points: Array.isArray(series.points) ? series.points.map((point) => ({ date: String(point.date || ""), value: chartValue(point.value) })).filter((point) => Number.isFinite(point.value)) : [],
  })).filter((series) => series.points.length);
}
function chartHeader(chart, fallbackLabel, unit) {
  const title = String(chart.title || fallbackLabel || t("chart"));
  const valuation = chart.valuation === "market-value" ? t("marketValue") : chart.valuation === "at-cost" ? t("atCost") : "";
  const period = chart.period || chartPeriodLabel();
  const description = [title, unit, period, valuation].filter(Boolean).join(" · ");
  return { title, period, valuation, description };
}
function chartAvailabilityNote(chart) {
  if (!chart || !chart.availability) return "";
  if (chart.availability === "unavailable-price") return `<p class="warning">${escapeHTML(t("unavailablePrice"))}</p>`;
  if (chart.availability === "unavailable-currency") return `<p class="warning">${escapeHTML(t("unavailableCurrency"))}</p>`;
  if (chart.availability === "native-multi") return `<p class="warning">${escapeHTML(t("notValued"))}</p>`;
  return "";
}
// renderTimeSeriesChart lays out a real value axis (left), date axis (bottom),
// zero line, and grid, keeps the first/last points through sampling, and
// exposes per-series toggles via data-series-index so wireCharts can bind them.
function renderTimeSeriesChart(chart, fallbackLabel) {
  const series = chartSeriesPoints(chart);
  if (!series.length) return "";
  if (chart.kind === "bar" || chart.kind === "stacked-bar") return renderTimeSeriesBars(chart, series, fallbackLabel);
  const unit = chart.currency || chart.unit || "";
  const words = chartHeader(chart, fallbackLabel, unit);
  // Sample the series so dense ledgers stay legible while the last point (the
  // current balance) is always present.
  const sampled = series.map((item) => ({ ...item, points: samplePoints(item.points) }));
  const pointCount = Math.max(...sampled.map((item) => item.points.length));
  const allValues = sampled.flatMap((item) => item.points.map((point) => point.value));
  const min = Math.min(0, ...allValues);
  const max = Math.max(0, ...allValues);
  const span = max - min || 1;
  const margin = { top: 4, right: 4, bottom: 8, left: 10 };
  const plotW = 100 - margin.left - margin.right;
  const plotH = 52 - margin.top - margin.bottom;
  const yFor = (value) => margin.top + plotH - ((value - min) / span) * plotH;
  const xFor = (index) => pointCount <= 1 ? margin.left + plotW / 2 : margin.left + (index / (pointCount - 1)) * plotW;
  // Y ticks on the left, horizontal grid lines across the plot.
  const yTicks = ticks(min, max).filter((value) => value >= min && value <= max);
  const gridLines = yTicks.map((value) => {
    const y = yFor(value);
    return `<line class="chart-grid" x1="${margin.left.toFixed(2)}" y1="${y.toFixed(2)}" x2="${(margin.left + plotW).toFixed(2)}" y2="${y.toFixed(2)}"></line><text class="chart-tick" x="${(margin.left - .6).toFixed(2)}" y="${(y + .4).toFixed(2)}" text-anchor="end">${escapeHTML(formatCompact(value))}</text>`;
  }).join("");
  const zeroY = yFor(0);
  const zeroLine = `<line class="chart-zero" x1="${margin.left.toFixed(2)}" y1="${zeroY.toFixed(2)}" x2="${(margin.left + plotW).toFixed(2)}" y2="${zeroY.toFixed(2)}"></line>`;
  const xTicks = dateTicks(sampled[0].points.map((point) => point.date));
  const xLabels = xTicks.map((date) => {
    const index = sampled[0].points.findIndex((point) => point.date === date);
    if (index < 0) return "";
    const x = xFor(index);
    return `<text class="chart-tick chart-tick-x" x="${x.toFixed(2)}" y="${(52 - 1.6).toFixed(2)}" text-anchor="middle">${escapeHTML(date)}</text>`;
  }).join("");
  const paths = sampled.map((item, seriesIndex) => {
    const points = item.points.map((point, index) => `${xFor(index).toFixed(2)},${yFor(point.value).toFixed(2)}`).join(" ");
    const circles = item.points.map((point, index) => `<circle class="series-${seriesIndex}" data-series-index="${seriesIndex}" data-point-date="${escapeHTML(point.date)}" data-point-value="${escapeHTML(String(point.value))}" cx="${xFor(index).toFixed(2)}" cy="${yFor(point.value).toFixed(2)}" r="1.1" tabindex="0" aria-label="${escapeHTML(`${item.label} ${point.date}: ${point.value}`)}"><title>${escapeHTML(`${item.label} ${point.date}: ${point.value}`)}</title></circle>`).join("");
    return `<polyline class="series-${seriesIndex}" points="${points}" fill="none" stroke-width="1.2"></polyline>${circles}`;
  }).join("");
  const legend = `<ol class="chart-legend">${sampled.map((item, index) => `<li data-series-toggle="${index}"><span class="series-label series-${index}">${escapeHTML(item.label)}</span><strong>${escapeHTML(`${item.points[item.points.length - 1].date}: ${item.points[item.points.length - 1].value}${unit ? ` ${unit}` : ""}`)}</strong></li>`).join("")}</ol>`;
  return `<section class="chart-card" aria-label="${escapeHTML(`${t("chart")}: ${words.description}`)}"><h3>${escapeHTML(words.title)}</h3>${chartAvailabilityNote(chart)}<svg class="report-chart report-line-chart" viewBox="0 0 100 52" role="img" aria-label="${escapeHTML(words.description)}" data-chart-unit="${escapeHTML(unit)}" data-chart-period="${escapeHTML(words.period)}">${gridLines}${zeroLine}${paths}${xLabels}</svg><p class="muted">${escapeHTML(words.description)}</p>${legend}</section>`;
}
function renderTimeSeriesBars(chart, series, fallbackLabel) {
  const unit = chart.currency || chart.unit || "";
  const words = chartHeader(chart, fallbackLabel, unit);
  const sampled = series.map((item) => ({ ...item, points: samplePoints(item.points) }));
  const pointCount = Math.max(...sampled.map((item) => item.points.length));
  const allValues = sampled.flatMap((item) => item.points.map((point) => point.value));
  const min = Math.min(0, ...allValues);
  const max = Math.max(0, ...allValues);
  const span = max - min || 1;
  const margin = { top: 4, right: 4, bottom: 8, left: 10 };
  const plotW = 100 - margin.left - margin.right;
  const plotH = 52 - margin.top - margin.bottom;
  const yFor = (value) => margin.top + plotH - ((value - min) / span) * plotH;
  const zeroY = yFor(0);
  const yTicks = ticks(min, max).filter((value) => value >= min && value <= max);
  const gridLines = yTicks.map((value) => {
    const y = yFor(value);
    return `<line class="chart-grid" x1="${margin.left.toFixed(2)}" y1="${y.toFixed(2)}" x2="${(margin.left + plotW).toFixed(2)}" y2="${y.toFixed(2)}"></line><text class="chart-tick" x="${(margin.left - .6).toFixed(2)}" y="${(y + .4).toFixed(2)}" text-anchor="end">${escapeHTML(formatCompact(value))}</text>`;
  }).join("");
  const zeroLine = `<line class="chart-zero" x1="${margin.left.toFixed(2)}" y1="${zeroY.toFixed(2)}" x2="${(margin.left + plotW).toFixed(2)}" y2="${zeroY.toFixed(2)}"></line>`;
  const groupWidth = pointCount ? plotW / pointCount : plotW;
  const stacked = sampled.some((item) => item.stacked);
  const bars = sampled.map((item, seriesIndex) => {
    return item.points.map((point, index) => {
      const valueY = yFor(point.value);
      let y = point.value < 0 ? zeroY : valueY;
      let height = Math.max(1, Math.abs(zeroY - valueY));
      let x = margin.left + index * groupWidth + seriesIndex * (groupWidth / Math.max(sampled.length, 1)) + (groupWidth / Math.max(sampled.length, 1) - groupWidth / Math.max(sampled.length, 1) * .72) / 2;
      let width = groupWidth / Math.max(sampled.length, 1) * .72;
      let dataPoints = Math.abs(point.value);
      if (stacked) {
        // Stack positive and negative contributions separately around zero.
        const prior = sampled.slice(0, seriesIndex).reduce((sum, prev) => {
          const prevPoint = prev.points[index];
          return sum + (prevPoint && Math.sign(prevPoint.value) === Math.sign(point.value) ? prevPoint.value : 0);
        }, 0);
        const base = point.value < 0 ? zeroY + ((prior) / span) * plotH : zeroY - ((prior) / span) * plotH;
        const top = point.value < 0 ? base + (point.value / span) * plotH : base - (point.value / span) * plotH;
        y = Math.min(base, top);
        height = Math.max(1, Math.abs(base - top));
        dataPoints = Math.abs(point.value);
        width = groupWidth * .8;
        x = margin.left + index * groupWidth + (groupWidth - width) / 2;
      }
      return `<rect class="series-${seriesIndex}${point.value < 0 ? " negative" : ""}" data-series-index="${seriesIndex}" data-point-date="${escapeHTML(point.date)}" data-point-value="${escapeHTML(String(point.value))}" x="${x.toFixed(2)}" y="${y.toFixed(2)}" width="${width.toFixed(2)}" height="${height.toFixed(2)}" rx=".6" tabindex="0" aria-label="${escapeHTML(`${item.label} ${point.date}: ${point.value}`)}"><title>${escapeHTML(`${item.label} ${point.date}: ${point.value}`)}</title></rect>`;
    }).join("");
  }).join("");
  const xTicks = dateTicks(sampled[0].points.map((point) => point.date));
  const xLabels = xTicks.map((date) => {
    const index = sampled[0].points.findIndex((point) => point.date === date);
    if (index < 0) return "";
    const x = margin.left + index * groupWidth + groupWidth / 2;
    return `<text class="chart-tick chart-tick-x" x="${x.toFixed(2)}" y="${(52 - 1.6).toFixed(2)}" text-anchor="middle">${escapeHTML(date)}</text>`;
  }).join("");
  const legend = `<ol class="chart-legend">${sampled.map((item, index) => `<li data-series-toggle="${index}"><span class="series-label series-${index}">${escapeHTML(item.label)}</span><strong>${escapeHTML(`${item.points[item.points.length - 1].date}: ${item.points[item.points.length - 1].value}${unit ? ` ${unit}` : ""}`)}</strong></li>`).join("")}</ol>`;
  return `<section class="chart-card" aria-label="${escapeHTML(`${t("chart")}: ${words.description}`)}"><h3>${escapeHTML(words.title)}</h3>${chartAvailabilityNote(chart)}<svg class="report-chart report-bar-chart" viewBox="0 0 100 52" role="img" aria-label="${escapeHTML(words.description)}" data-chart-unit="${escapeHTML(unit)}" data-chart-period="${escapeHTML(words.period)}">${gridLines}${zeroLine}${bars}${xLabels}</svg><p class="muted">${escapeHTML(words.description)}</p>${legend}</section>`;
}
// hierarchyLayoutSvg builds a treemap, sunburst, or icicle from the flattened
// chart.nodes tree. Each node becomes a focusable shape whose click drills into
// the account page. Layout math is kept minimal and dependency-free.
function flattenHierarchy(chart) {
  const flat = [];
  const walk = (nodes) => {
    (nodes || []).forEach((node) => {
      // node.value is a serialized PresentedDecimal ({display, exact}) or a
      // plain number/string; normalize it so layout math sees a number.
      const value = chartValue(node != null ? node.value : null);
      flat.push({ ...node, value });
      walk(node.children);
    });
  };
  walk(chart && chart.nodes);
  let computed = flat;
  if (!computed.length && chart && Array.isArray(chart.series)) {
    // Fallback for a hierarchy chart with a flat series payload: represent
    // each series as a node so the layout still renders.
    computed = chart.series.map((series) => ({ name: series.label || "", currency: chart.currency || "", value: chartValue(series.points && series.points.length ? series.points[series.points.length - 1].value : 0), depth: 0, children: [] }));
  }
  return computed;
}
function hierarchyTotal(nodes) {
  return (nodes || []).reduce((sum, node) => sum + Math.abs(Number(node.value || 0)), 0);
}
// layoutTreemapRects recursively bisects a rectangle: items are split into two
// value-balanced groups, the group boundary cuts along the rectangle's longer
// side, and each group recurses into its share of the area. This keeps every
// leaf's rectangle proportional to its value while staying reliably
// two-dimensional (the previous single-row sqrt-width layout degenerated into
// a thin sliver of near-zero-width rectangles once leaf values had a wide
// spread, which is the common case for a real multi-account ledger).
function layoutTreemapRects(items, x, y, w, h) {
  if (!items.length || w <= 0 || h <= 0) return [];
  if (items.length === 1) return [{ node: items[0].node, x, y, w, h }];
  const total = items.reduce((sum, item) => sum + item.value, 0);
  if (!total) return [];
  let running = 0;
  let splitIndex = 1;
  for (let index = 0; index < items.length; index++) {
    running += items[index].value;
    if (running >= total / 2) { splitIndex = index + 1; break; }
  }
  splitIndex = Math.min(Math.max(splitIndex, 1), items.length - 1);
  const groupA = items.slice(0, splitIndex);
  const groupB = items.slice(splitIndex);
  const fractionA = groupA.reduce((sum, item) => sum + item.value, 0) / total;
  if (w >= h) {
    const widthA = w * fractionA;
    return [...layoutTreemapRects(groupA, x, y, widthA, h), ...layoutTreemapRects(groupB, x + widthA, y, w - widthA, h)];
  }
  const heightA = h * fractionA;
  return [...layoutTreemapRects(groupA, x, y, w, heightA), ...layoutTreemapRects(groupB, x, y + heightA, w, h - heightA)];
}
function renderHierarchyTreemap(chart) {
  const nodes = flattenHierarchy(chart);
  const leaves = [];
  const addLeaves = (n) => {
    if (n.children && n.children.length) n.children.forEach(addLeaves);
    else leaves.push(n);
  };
  nodes.forEach(addLeaves);
  const items = leaves
    .map((node) => ({ node, value: Math.abs(Number(node.value || 0)) }))
    .filter((item) => item.value > 0)
    .sort((a, b) => b.value - a.value);
  if (!items.length) return "";
  const gap = 0.3;
  return layoutTreemapRects(items, 0, 4, 100, 44).map(({ node, x, y, w, h }) => {
    const rx = x + gap / 2, ry = y + gap / 2;
    const rw = Math.max(0.4, w - gap), rh = Math.max(0.4, h - gap);
    return `<rect class="hierarchy-node series-${(node.depth || 0) % 4}" data-account="${escapeHTML(node.name || "")}" x="${rx.toFixed(2)}" y="${ry.toFixed(2)}" width="${rw.toFixed(2)}" height="${rh.toFixed(2)}" rx=".5" tabindex="0" aria-label="${escapeHTML(`${node.name || ""}: ${node.value || 0} ${chart.currency || ""}`)}"><title>${escapeHTML(`${node.name || ""}: ${node.value || 0} ${chart.currency || ""}`)}</title></rect>`;
  }).join("");
}
function renderHierarchySunburst(chart) {
  const nodes = flattenHierarchy(chart);
  const total = hierarchyTotal(nodes);
  if (!total) return "";
  const maxDepth = Math.max(0, ...nodes.map((node) => node.depth || 0));
  const ringStep = maxDepth ? 44 / (maxDepth + 1) : 44;
  let cursor = 0;
  return nodes.map((node) => {
    const fraction = Math.abs(Number(node.value || 0)) / total;
    const start = cursor;
    cursor += fraction * 360;
    const cx = 50;
    const base = 4 + (node.depth || 0) * ringStep;
    const outer = Math.max(base + .5, (node.depth || 0) * ringStep + ringStep);
    const startRad = start * Math.PI / 180;
    const endRad = cursor * Math.PI / 180;
    const large = endRad - startRad > Math.PI ? 1 : 0;
    const x1 = cx + base * Math.cos(startRad);
    const y1 = 26 + base * Math.sin(startRad);
    const x2 = cx + outer * Math.cos(startRad);
    const y2 = 26 + outer * Math.sin(startRad);
    const x3 = cx + outer * Math.cos(endRad);
    const y3 = 26 + outer * Math.sin(endRad);
    const x4 = cx + base * Math.cos(endRad);
    const y4 = 26 + base * Math.sin(endRad);
    return `<path class="hierarchy-node series-${(node.depth || 0) % 4}" data-account="${escapeHTML(node.name || "")}" d="M${x1.toFixed(2)} ${y1.toFixed(2)} L${x2.toFixed(2)} ${y2.toFixed(2)} A${outer.toFixed(2)} ${outer.toFixed(2)} 0 ${large} 1 ${x3.toFixed(2)} ${y3.toFixed(2)} L${x4.toFixed(2)} ${y4.toFixed(2)} A${base.toFixed(2)} ${base.toFixed(2)} 0 ${large} 0 ${x1.toFixed(2)} ${y1.toFixed(2)} Z" tabindex="0" aria-label="${escapeHTML(`${node.name || ""}: ${node.value || 0} ${chart.currency || ""}`)}"><title>${escapeHTML(`${node.name || ""}: ${node.value || 0} ${chart.currency || ""}`)}</title></path>`;
  }).join("");
}
function renderHierarchyIcicle(chart) {
  const nodes = flattenHierarchy(chart);
  const total = hierarchyTotal(nodes);
  if (!total) return "";
  const maxDepth = Math.max(0, ...nodes.map((node) => node.depth || 0));
  const rowHeight = maxDepth ? 44 / (maxDepth + 1) : 44;
  let cursor = 0;
  return nodes.map((node) => {
    const width = Math.abs(Number(node.value || 0)) / total * 100;
    const y = 4 + (node.depth || 0) * rowHeight;
    return `<rect class="hierarchy-node series-${(node.depth || 0) % 4}" data-account="${escapeHTML(node.name || "")}" x="${cursor.toFixed(2)}" y="${y.toFixed(2)}" width="${Math.max(1, width).toFixed(2)}" height="${Math.max(2, rowHeight - 1).toFixed(2)}" rx="1" tabindex="0" aria-label="${escapeHTML(`${node.name || ""}: ${node.value || 0} ${chart.currency || ""}`)}"><title>${escapeHTML(`${node.name || ""}: ${node.value || 0} ${chart.currency || ""}`)}</title></rect>`;
  }).join("");
}
function renderHierarchyChart(chart, fallbackLabel) {
  const unit = chart.currency || chart.unit || "";
  const words = chartHeader(chart, fallbackLabel, unit);
  const layout = reportState.hierarchyLayout === "sunburst" ? renderHierarchySunburst(chart) : reportState.hierarchyLayout === "icicle" ? renderHierarchyIcicle(chart) : renderHierarchyTreemap(chart);
  if (!layout) return `<section class="chart-card" aria-label="${escapeHTML(`${t("chart")}: ${words.description}`)}"><h3>${escapeHTML(words.title)}</h3>${chartAvailabilityNote(chart)}<p class="warning">${escapeHTML(t("noConversionData"))}</p></section>`;
  return `<section class="chart-card" aria-label="${escapeHTML(`${t("chart")}: ${words.description}`)}"><h3>${escapeHTML(words.title)}</h3>${chartAvailabilityNote(chart)}<svg class="report-chart report-hierarchy-chart" viewBox="0 0 100 52" role="img" aria-label="${escapeHTML(words.description)}" data-chart-unit="${escapeHTML(unit)}" data-chart-period="${escapeHTML(words.period)}">${layout}</svg><p class="muted">${escapeHTML(words.description)}</p></section>`;
}
function renderChart(result, label, route, chart) {
  if (chart && chart.kind === "hierarchy") return renderHierarchyChart(chart, label);
  if (chart && Array.isArray(chart.series) && chart.series.some((series) => Array.isArray(series.points) && series.points.length)) return renderTimeSeriesChart(chart, label);
  const { points, title, unit, period } = semanticChartPoints(result, route);
  if (!points.length) return "";
  const max = Math.max(...points.map((point) => Math.abs(point.value)), 1);
  const barWidth = 100 / points.length;
  const bars = points.map((point, index) => {
    const height = Math.max(1, Math.round(Math.abs(point.value) / max * 45));
    const x = (index * barWidth + barWidth * .12).toFixed(2);
    const width = (barWidth * .72).toFixed(2);
    const y = point.value < 0 ? 50 : 50 - height;
    const className = point.value < 0 ? "negative" : "";
    return `<rect class="${className}" x="${x}%" y="${y}%" width="${width}%" height="${height}%" rx="1" tabindex="0" aria-label="${escapeHTML(`${point.label}: ${point.value} ${unit || ""}`)}"><title>${escapeHTML(`${point.label}: ${point.value} ${unit || ""}`)}</title></rect>`;
  }).join("");
  const description = `${title || label} · ${unit || t("unavailable")} · ${period || chartPeriodLabel()}`;
  const legend = `<ol class="chart-legend">${points.map((point) => `<li><span>${escapeHTML(point.label)}</span><strong>${escapeHTML(String(point.value))}${unit ? ` ${escapeHTML(unit)}` : ""}</strong></li>`).join("")}</ol>`;
  return `<section class="chart-card" aria-label="${escapeHTML(`${t("chart")}: ${description}`)}"><h3>${escapeHTML(title || label)}</h3><svg class="report-chart" viewBox="0 0 100 52" role="img" aria-label="${escapeHTML(description)}" data-chart-unit="${escapeHTML(unit || "")}" data-chart-period="${escapeHTML(period || chartPeriodLabel())}"><line x1="0" y1="50" x2="100" y2="50" class="chart-axis"></line>${bars}</svg><p class="muted">${escapeHTML(description)}</p>${legend}</section>`;
}
function reportToolbar(route) {
  const exportURL = `${reportURL(route)}${reportURL(route).includes("?") ? "&" : "?"}format=csv`;
  const asOf = route === "holdings" ? `<label>${escapeHTML(t("to"))} <input id="report-as-of" type="date" value="${escapeHTML(reportState.asOf)}"></label>` : "";
  const isTree = route === "accounts" || route === "trial-balance" || route === "balance-sheet" || route === "income-statement";
  const treeToggle = isTree ? `<label><input id="report-tree-mode" type="checkbox" ${reportState.treeMode === "leaves" ? "checked" : ""}> ${escapeHTML(t("leavesOnly"))}</label>` : "";
  const layout = route === "trial-balance" ? `<label>${escapeHTML(t("hierarchy"))} <select id="chart-hierarchy-layout"><option value="treemap">${escapeHTML(t("treemap"))}</option><option value="sunburst">${escapeHTML(t("sunburst"))}</option><option value="icicle">${escapeHTML(t("icicle"))}</option></select></label>` : "";
  return `<div class="toolbar report-toolbar" role="group" aria-label="${escapeHTML(t("period"))}"><label>${escapeHTML(t("period"))} <select id="report-period"><option value="all">${escapeHTML(t("allPeriods"))}</option><option value="month">${escapeHTML(t("monthly"))}</option><option value="quarter">${escapeHTML(t("quarterly"))}</option><option value="year">${escapeHTML(t("yearly"))}</select></label><label>${escapeHTML(t("valuation"))} <select id="report-valuation"><option value="at-cost">${escapeHTML(t("atCost"))}</option><option value="market-value">${escapeHTML(t("marketValue"))}</option></select></label>${layout}${treeToggle}${asOf}<a class="button" id="report-export" href="${escapeHTML(exportURL)}" download>${escapeHTML(t("exportCSV"))}</a></div>`;
}
function wireReportToolbar(route) {
  const period = document.getElementById("report-period");
  const valuation = document.getElementById("report-valuation");
  if (!period || !valuation) return;
  period.value = reportState.period;
  valuation.value = reportState.valuation;
  const layout = document.getElementById("chart-hierarchy-layout");
  if (layout) layout.value = reportState.hierarchyLayout;
  const treeMode = document.getElementById("report-tree-mode");
  if (treeMode) treeMode.checked = reportState.treeMode === "leaves";
  const apply = () => {
    reportState.period = period.value || "all";
    reportState.valuation = valuation.value || "at-cost";
    const asOf = document.getElementById("report-as-of");
    reportState.asOf = asOf ? asOf.value : reportState.asOf;
    if (layout) reportState.hierarchyLayout = layout.value;
    if (treeMode) reportState.treeMode = treeMode.checked ? "leaves" : "hierarchy";
    updateURL();
    renderReport(route);
  };
  period.addEventListener("change", apply);
  valuation.addEventListener("change", apply);
  if (layout) layout.addEventListener("change", apply);
  if (treeMode) treeMode.addEventListener("change", apply);
  const asOf = document.getElementById("report-as-of");
  if (asOf) asOf.addEventListener("change", apply);
}
function renderError(error) { return `<p class="error">${escapeHTML(t("requestFailed"))}: ${escapeHTML(error.message || error)}</p>`; }
function api(path) {
  return fetch(path, { headers: { Accept: "application/json" } }).then(async (response) => {
    const body = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(body.error || `${response.status} ${response.statusText}`);
    return body;
  });
}
async function apiJSON(path, method, payload) {
  const response = await fetch(path, { method, headers: { Accept: "application/json", "Content-Type": "application/json" }, body: JSON.stringify(payload) });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(body.error || `${response.status} ${response.statusText}`);
  return body;
}
function updateURL() {
  const next = new URL(window.location.href);
  next.searchParams.set("view", view);
  next.searchParams.set("time", globalState.time);
  next.searchParams.set("currency", globalState.currency);
  next.searchParams.set("period", reportState.period);
  next.searchParams.set("valuation", reportState.valuation);
  if (reportState.treeMode === "leaves") next.searchParams.set("tree_mode", "leaves"); else next.searchParams.delete("tree_mode");
  if (reportState.hierarchyLayout && reportState.hierarchyLayout !== "treemap") next.searchParams.set("hierarchy_layout", reportState.hierarchyLayout); else next.searchParams.delete("hierarchy_layout");
  if (reportState.asOf) next.searchParams.set("as_of", reportState.asOf); else next.searchParams.delete("as_of");
  if (globalState.account) next.searchParams.set("account", globalState.account); else next.searchParams.delete("account");
  if (globalState.filter) next.searchParams.set("filter", globalState.filter); else next.searchParams.delete("filter");
  if (view === "journal") {
    if (journalRange.from) next.searchParams.set("from", journalRange.from); else next.searchParams.delete("from");
    if (journalRange.to) next.searchParams.set("to", journalRange.to); else next.searchParams.delete("to");
    ["flag", "tag", "link", "payee", "narration", "kind"].forEach((key) => {
      if (journalFilters[key]) next.searchParams.set(key, journalFilters[key]); else next.searchParams.delete(key);
    });
    if (journalTableState.sort) {
      next.searchParams.set("sort", journalTableState.sort);
      next.searchParams.set("direction", journalTableState.direction || "ascending");
    } else {
      next.searchParams.delete("sort");
      next.searchParams.delete("direction");
    }
    if (journalTableState.page > 0) next.searchParams.set("page", String(journalTableState.page + 1)); else next.searchParams.delete("page");
    if (journalWindow !== "month") next.searchParams.set("window", journalWindow); else next.searchParams.delete("window");
    if (journalOrder === "asc") next.searchParams.set("order", "asc"); else next.searchParams.delete("order");
  } else {
    ["from", "to", "flag", "tag", "link", "payee", "narration", "kind", "sort", "direction", "page", "window", "order"].forEach((key) => next.searchParams.delete(key));
  }
  if (chartWindowState !== "1y") next.searchParams.set("chart_window", chartWindowState); else next.searchParams.delete("chart_window");
  window.history.replaceState({}, "", next);
}
function globalQuery() {
  const search = new URLSearchParams();
  if (globalState.time && globalState.time !== "all") search.set("time", globalState.time);
  if (globalState.account) search.set("account", globalState.account);
  if (globalState.filter) search.set("filter", globalState.filter);
  if (globalState.currency) search.set("currency", globalState.currency);
  return search;
}
const chartWindowChoices = [["3m", "chartWindow3m"], ["6m", "chartWindow6m"], ["1y", "chartWindow1y"], ["3y", "chartWindow3y"], ["all", "chartWindowAll"]];
function chartWindowMonths(key) { return { "3m": 3, "6m": 6, "1y": 12, "3y": 36 }[key] || 0; }
function chartsHaveTimeSeries(charts) {
  return (charts || []).some((chart) => chart && Array.isArray(chart.series) && chart.series.some((series) => Array.isArray(series.points) && series.points.some((point) => point.date)));
}
// sliceChartsByWindow narrows every time series to the selected time range,
// anchored on the latest point across the report's charts (FD-0008). Balance
// charts are cumulative, so the first surviving point still reads as a level.
function sliceChartsByWindow(charts) {
  const months = chartWindowMonths(chartWindowState);
  if (!months || !chartsHaveTimeSeries(charts)) return charts;
  const anchor = (charts || []).flatMap((chart) => chart.series || []).flatMap((series) => series.points || []).map((point) => point.date).filter(Boolean).sort().pop();
  if (!anchor) return charts;
  const cutoffDate = new Date(`${anchor}T00:00:00Z`);
  cutoffDate.setUTCMonth(cutoffDate.getUTCMonth() - months);
  const cutoff = cutoffDate.toISOString().slice(0, 10);
  return charts.filter(Boolean).map((chart) => ({ ...chart, series: (chart.series || []).map((series) => ({ ...series, points: (series.points || []).filter((point) => point.date >= cutoff) })) }));
}
function chartWindowBar(show) {
  if (!show) return "";
  return `<div class="toolbar chart-window-bar" role="group" aria-label="${escapeHTML(t("chartWindow"))}"><label>${escapeHTML(t("chartWindow"))} <select id="chart-window">${chartWindowChoices.map(([value, key]) => `<option value="${value}" ${chartWindowState === value ? "selected" : ""}>${escapeHTML(t(key))}</option>`).join("")}</select></label></div>`;
}
function wireChartWindow() {
  const select = document.getElementById("chart-window");
  if (!select) return;
  select.addEventListener("change", (event) => { chartWindowState = event.target.value; updateURL(); render(); });
}
function reportURL(route) {
  const search = globalQuery();
  // Holdings and the auxiliary reports are complete, independent views. A
  // stale top-bar account/text/time filter used to be sent to every endpoint;
  // rows such as prices and events have no account field, so one old filter
  // made several otherwise-populated pages look empty at once.
  if (["holdings", "commodities", "prices", "events", "documents", "statistics"].includes(route)) {
    ["time", "account", "filter"].forEach((key) => search.delete(key));
  }
  const apiRoute = route === "commodities" ? "prices" : route;
  if (reportState.period && reportState.period !== "all") search.set("period", reportState.period);
  if (reportState.valuation && reportState.valuation !== "at-cost") search.set("valuation", reportState.valuation);
  if (reportState.asOf && route === "holdings") search.set("as_of", reportState.asOf);
  // The income statement and balance sheet charts plot one series per
  // currency natively, so they need no display currency; forcing one would
  // request conversion quotes the ledger may not have. Trial balance and
  // account drilldowns still convert and keep the default.
  if (!search.has("currency") && (route === "trial-balance" || route === "accounts")) search.set("currency", effectiveCurrency());
  const suffix = search.toString();
  return `/api/v1/reports/${encodeURIComponent(apiRoute)}${suffix ? `?${suffix}` : ""}`;
}
// effectiveCurrency is the active chart display currency: the explicit global
// preference when set, otherwise the ledger operating currency, otherwise USD.
function effectiveCurrency() {
  return globalState.currency || operatingCurrency || "USD";
}
function applyGlobalState() {
  globalState.time = timePicker.value || "all";
  globalState.account = accountPicker.value || "";
  globalState.filter = globalFilter.value.trim();
  updateURL();
  render();
}
function renderNavigation() {
  const coreRoutes = navRoutes.filter(([route]) => route !== "diagnostics" && route !== "source" && route !== "accounts" && route !== "overview");
  const extendedRoutes = navRoutes.filter(([route]) => route === "overview" || route === "accounts" || route === "source");
  const diagnosticsVisible = diagnosticCount > 0 || view === "diagnostics";
  const core = coreRoutes.map(([route, key]) => `<button type="button" data-view="${route}" aria-current="${view === route ? "page" : "false"}">${escapeHTML(t(key))}</button>`).join("");
  const extended = extendedRoutes.map(([route, key]) => `<button type="button" data-view="${route}" aria-current="${view === route ? "page" : "false"}">${escapeHTML(t(key))}</button>`).join("");
  const errors = diagnosticsVisible ? `<button type="button" data-view="diagnostics" aria-current="${view === "diagnostics" ? "page" : "false"}">${escapeHTML(t("diagnostics"))}${diagnosticCount ? ` (${diagnosticCount})` : ""}</button>` : "";
  navigation.innerHTML = `${core}${errors}${extended ? `<div class="sidebar-heading">${escapeHTML(t("extended"))}</div>${extended}` : ""}`;
  navigation.querySelectorAll("button[data-view]").forEach((button) => button.addEventListener("click", () => {
    if (button.dataset.view !== "help") helpPage = "";
    view = button.dataset.view;
    updateURL();
    sidebar.classList.remove("open");
    menuToggle.setAttribute("aria-expanded", "false");
    render();
  }));
}
async function renderOverview() {
  app.innerHTML = `<p class="muted">${escapeHTML(t("loading"))}</p>`;
  try {
    const status = await api(`/api/v1/status?locale=${encodeURIComponent(locale)}`);
    app.innerHTML = `<div class="cards"><section class="card">${escapeHTML(t("snapshot"))}<strong><code>${escapeHTML(status.snapshot_id || "—")}</code></strong></section><section class="card">${escapeHTML(t("valid"))}<strong>${status.valid ? escapeHTML(t("yes")) : escapeHTML(t("no"))}</strong></section><section class="card">${escapeHTML(t("accountsCount"))}<strong>${status.account_count || 0}</strong></section><section class="card">${escapeHTML(t("diagnosticsCount"))}<strong>${status.diagnostic_count || 0}</strong></section><section class="card">${escapeHTML(t("publishedAt"))}<strong>${escapeHTML(status.published_at || "—")}</strong></section></div>`;
  } catch (error) { app.innerHTML = renderError(error); }
}
const journalEntryBadges = [
  ["kind", "transaction", "Transaction"], ["flag", "*", "*"], ["flag", "!", "!"],
];
const journalWindowOptions = [["all", "windowAll"], ["week", "windowWeek"], ["month", "windowMonth"], ["quarter", "windowQuarter"], ["year", "windowYear"], ["custom", "windowCustom"]];
// journalWindowFromISO anchors the window on the ledger's latest entry date
// (not today), so a ledger paused months ago still opens on its most recent
// month instead of an empty screen. Returns "" when no window applies.
function journalWindowFromISO() {
  if (!journalAnchorDate || journalWindow === "custom" || journalWindow === "all") return "";
  const date = new Date(`${journalAnchorDate}T00:00:00Z`);
  const days = { week: 7 }[journalWindow];
  const months = { month: 1, quarter: 3, year: 12 }[journalWindow] || 0;
  if (days) date.setUTCDate(date.getUTCDate() - days);
  else date.setUTCMonth(date.getUTCMonth() - months);
  return date.toISOString().slice(0, 10);
}
function journalToolbar() {
  const badges = journalEntryBadges.map(([type, value, label]) => {
    const active = type === "kind" ? journalFilters.kind === value : journalFilters.flag === value;
    return `<button type="button" class="journal-entry-badge${active ? " badge-on" : ""}" data-journal-type="${escapeHTML(type)}" data-journal-value="${escapeHTML(value)}" aria-pressed="${active ? "true" : "false"}">${escapeHTML(t(label))}</button>`;
  }).join("");
  return `<div class="toolbar journal-toolbar" role="group" aria-label="${escapeHTML(t("journal"))}"><div class="journal-entry-badges" role="group" aria-label="${escapeHTML(t("postings"))}">${badges}</div><label>${escapeHTML(t("from"))} <input id="journal-from" type="date" value="${escapeHTML(journalRange.from)}"></label><label>${escapeHTML(t("to"))} <input id="journal-to" type="date" value="${escapeHTML(journalRange.to)}"></label><label>${escapeHTML(t("windowLabel"))} <select id="journal-window">${journalWindowOptions.map(([value, key]) => `<option value="${value}" ${journalWindow === value ? "selected" : ""}>${escapeHTML(t(key))}</option>`).join("")}</select></label><button id="journal-order" type="button" aria-pressed="${journalOrder === "desc" ? "true" : "false"}">${escapeHTML(t(journalOrder === "desc" ? "orderNewest" : "orderOldest"))}</button><label>${escapeHTML(t("flag"))} <input id="journal-flag" type="text" inputmode="text" value="${escapeHTML(journalFilters.flag)}" placeholder="* or !"></label><label>${escapeHTML(t("tag"))} <input id="journal-tag" type="search" value="${escapeHTML(journalFilters.tag)}"></label><label>${escapeHTML(t("link"))} <input id="journal-link" type="search" value="${escapeHTML(journalFilters.link)}"></label><label>${escapeHTML(t("payee"))} <input id="journal-payee" type="search" value="${escapeHTML(journalFilters.payee)}"></label><label>${escapeHTML(t("narration"))} <input id="journal-narration" type="search" value="${escapeHTML(journalFilters.narration)}"></label><button id="journal-apply" type="button">${escapeHTML(t("apply"))}</button><button id="journal-reset" type="button">${escapeHTML(t("reset"))}</button><a class="button" id="journal-export" href="${escapeHTML(`${journalURL(true)}${journalURL(true).includes("?") ? "&" : "?"}format=csv`)}" download>${escapeHTML(t("exportCSV"))}</a></div>`;
}
function wireJournalEntryBadges() {
  document.querySelectorAll(".journal-entry-badge").forEach((button) => button.addEventListener("click", () => {
    const type = button.dataset.journalType;
    const value = button.dataset.journalValue;
    if (type === "kind") {
      journalFilters.kind = journalFilters.kind === value ? "" : value;
    } else {
      journalFilters.flag = journalFilters.flag === value ? "" : value;
    }
    updateURL();
    renderReport("journal");
  }));
}
function journalURL(forExport = false) {
  const search = globalQuery();
  if (journalRange.from) search.set("from", journalRange.from);
  if (journalRange.to) search.set("to", journalRange.to);
  // The window slices client-side for the screen; the CSV export carries the
  // equivalent from date so the downloaded file matches what is on screen.
  if (forExport && !journalRange.from) {
    const windowFrom = journalWindowFromISO();
    if (windowFrom) search.set("from", windowFrom);
  }
  Object.entries(journalFilters).forEach(([key, value]) => { if (value) search.set(key, value); });
  const suffix = search.toString();
  return `/api/v1/reports/journal${suffix ? `?${suffix}` : ""}`;
}
// renderJournal groups flat posting rows into transaction header + expandable
// posting detail rows. Postings of the same transaction share (file, span);
// header rows carry the date, payee/narration, and a toggle that reveals the
// indented posting rows beneath.
function renderJournal(result) {
  const container = document.getElementById("journal-result");
  let rows = Array.isArray(result.rows) ? result.rows : [];
  if (!rows.length) {
    container.innerHTML = `<p class="muted">${escapeHTML(t("empty"))}</p>`;
    return;
  }
  // The fetch arrives unfiltered (unless the user set explicit dates), so the
  // latest entry anchors the default window before slicing.
  journalAnchorDate = rows.reduce((max, row) => (row.date && (!max || row.date > max) ? row.date : max), "");
  const windowFrom = journalWindowFromISO();
  if (windowFrom) rows = rows.filter((row) => !row.date || row.date >= windowFrom);
  const exportLink = document.getElementById("journal-export");
  if (exportLink) exportLink.href = `${journalURL(true)}${journalURL(true).includes("?") ? "&" : "?"}format=csv`;
  const groups = [];
  const byKey = new Map();
  rows.forEach((row) => {
    const key = `${row.file || ""}::${row.span || ""}::${row.date || ""}`;
    if (!byKey.has(key)) {
      byKey.set(key, []);
      groups.push({ key, rows: byKey.get(key) });
    }
    byKey.get(key).push(row);
  });
  if (journalOrder === "desc") groups.reverse();
  const html = groups.map((group) => {
    const first = group.rows[0];
    const header = `<tr class="journal-transaction-row" data-journal-group="${escapeHTML(group.key)}"><td colspan="4"><button type="button" class="journal-transaction-toggle" aria-expanded="true" aria-label="${escapeHTML(`${t("details")}: ${first.date || ""}`)}">▾</button><span class="muted">${escapeHTML(first.date || "")}</span>${first.payee ? ` <strong>${escapeHTML(first.payee)}</strong>` : ""}${first.narration ? ` · ${escapeHTML(first.narration)}` : ""}<span class="muted"> · ${group.rows.length} ${escapeHTML(t("postings"))}</span></td></tr>`;
    const detail = group.rows.map((posting) => `<tr class="journal-posting-row" data-journal-group="${escapeHTML(group.key)}"><td></td><td>${escapeHTML(posting.account || "")}</td><td>${escapeHTML(postedUnits(posting))}</td><td>${escapeHTML(postedCost(posting))}</td></tr>`).join("");
    return header + detail;
  }).join("");
  container.innerHTML = `<div class="table-wrap"><table><thead><tr><th class="sr-only">${escapeHTML(t("details"))}</th><th>${escapeHTML(t("account"))}</th><th>${escapeHTML(t("units"))}</th><th>${escapeHTML(t("cost"))}</th></tr></thead><tbody>${html}</tbody></table></div>`;
  container.querySelectorAll(".journal-transaction-toggle").forEach((button) => button.addEventListener("click", () => {
    const key = button.dataset.journalGroup;
    const expanded = button.getAttribute("aria-expanded") === "true";
    button.setAttribute("aria-expanded", String(!expanded));
    button.textContent = expanded ? "▸" : "▾";
    container.querySelectorAll(`.journal-posting-row[data-journal-group="${CSS.escape(key)}"]`).forEach((row) => { row.hidden = expanded; });
  }));
}
function postedUnits(posting) {
  const units = posting.units;
  if (units && typeof units === "object") {
    const amount = units.display != null ? units.display : units;
    const currency = posting.currency || "";
    return `${amount}${currency ? ` ${currency}` : ""}`;
  }
  return posting.units != null ? String(posting.units) : "";
}
function postedCost(posting) {
  const cost = posting.cost;
  if (cost && typeof cost === "object") {
    const amount = cost.display != null ? cost.display : cost;
    return `${amount}${posting.cost_currency ? ` ${posting.cost_currency}` : ""}`;
  }
  return "";
}
async function renderReport(route) {
  app.innerHTML = `<p class="muted">${escapeHTML(t("loading"))}</p>`;
  try {
    const result = await api(route === "journal" ? journalURL() : reportURL(route));
    if (route === "journal") {
      app.innerHTML = `${journalToolbar()}<div id="journal-result"></div>`;
      document.getElementById("journal-apply").addEventListener("click", () => {
        journalRange = { from: document.getElementById("journal-from").value, to: document.getElementById("journal-to").value };
        ["flag", "tag", "link", "payee", "narration", "kind"].forEach((key) => { journalFilters[key] = document.getElementById(`journal-${key}`).value.trim(); });
        if (journalRange.from || journalRange.to) journalWindow = "custom";
        else if (journalWindow === "custom") journalWindow = "month";
        updateURL();
        renderReport("journal");
      });
      document.getElementById("journal-reset").addEventListener("click", () => { journalRange = { from: "", to: "" }; journalFilters = { flag: "", tag: "", link: "", payee: "", narration: "", kind: "" }; journalWindow = "month"; journalOrder = "desc"; updateURL(); renderReport("journal"); });
      document.getElementById("journal-window").addEventListener("change", (event) => {
        journalWindow = event.target.value;
        if (journalWindow !== "custom") journalRange = { from: "", to: "" };
        updateURL(); renderReport("journal");
      });
      document.getElementById("journal-order").addEventListener("click", () => {
        journalOrder = journalOrder === "desc" ? "asc" : "desc";
        updateURL(); renderReport("journal");
      });
      wireJournalEntryBadges();
      renderJournal(result);
      return;
    }
    const tree = route === "accounts" || route === "trial-balance" || route === "balance-sheet" || route === "income-statement";
    // Account detail view: when a single exact account is selected, render a
    // dedicated balance chart + posting detail (with running balance) instead
    // of the generic filtered accounts list.
    if (route === "accounts" && globalState.account) {
      await renderAccountDetail(globalState.account);
      return;
    }
    const asOfNote = route === "holdings" && reportState.asOf ? `<p class="muted">${escapeHTML(t("survivingLots"))}</p>` : "";
    // The statement trees carry a per-measure, per-currency chart set: one
    // chart per measure, one series per currency. Nothing is converted, so a
    // ledger without FX quotes plots every currency instead of warning.
    const reportLabel = t(navRoutes.find(([name]) => name === route)?.[1] || route);
    // Chart-less reports (holdings, prices, events, documents, statistics)
    // carry no chart payload; drop the empty placeholder so the chart window
    // helpers never see an undefined chart.
    const chartList = (Array.isArray(result.charts) && result.charts.length ? result.charts : [result.chart]).filter(Boolean);
    const windowed = sliceChartsByWindow(chartList);
    app.innerHTML = `${reportToolbar(route)}${asOfNote}${chartWindowBar(chartsHaveTimeSeries(chartList))}${windowed.map((chart) => renderChart(result, reportLabel, route, chart)).join("")}<div id="report-result"></div>`;
    wireReportToolbar(route);
    wireChartWindow();
    app.querySelectorAll(".report-chart").forEach((svg, index) => mountChartData(svg, windowed[index]));
    wireCharts(app);
    mountTable(document.getElementById("report-result"), result, { tree, leavesOnly: tree && reportState.treeMode === "leaves", pivotCurrency: tree });
  } catch (error) { app.innerHTML = renderError(error); }
}
// renderAccountDetail renders a dedicated account page: a balance time-series
// chart for the account plus its posting detail table with a running balance
// column (Fava's account journal).
async function renderAccountDetail(account) {
  app.innerHTML = `<p class="muted">${escapeHTML(t("loading"))}</p>`;
  try {
    const displayCurrency = globalState.currency || operatingCurrency || "";
    const [chartResult, journalResult] = await Promise.all([
      api(`/api/v1/reports/accounts?currency=${encodeURIComponent(displayCurrency)}&account=${encodeURIComponent(account)}&period=${encodeURIComponent(reportState.period || "all")}`),
      api(`/api/v1/reports/journal?account=${encodeURIComponent(account)}`),
    ]);
    const chart = sliceChartsByWindow([chartResult.chart])[0] || chartResult.chart;
    const head = `<div class="account-detail-head"><a href="?view=accounts">&larr; ${escapeHTML(t("back"))}</a><h3>${escapeHTML(account)}</h3></div>`;
    const chartHtml = chart && Array.isArray(chart.series) && chart.series.some((series) => Array.isArray(series.points) && series.points.length) ? `${chartWindowBar(true)}${renderChart(chartResult, t("accountBalance"), "accounts", chart)}` : `<p class="muted">${escapeHTML(t("empty"))}</p>`;
    const rows = Array.isArray(journalResult.rows) ? journalResult.rows : [];
    // Running balance is tracked per currency; summing across currencies would
    // produce a meaningless number for a multi-currency account.
    const runningByCurrency = new Map();
    const detailRows = rows.map((row) => {
      const amount = postedAmountValue(row);
      const currency = row.currency || "";
      let running = "";
      if (amount != null && currency) {
        const current = runningByCurrency.get(currency) || { exact: "0", display: "0" };
        const next = addExact(current.exact, amount.exact);
        runningByCurrency.set(currency, { exact: String(next.exact), display: String(next.display) });
        running = String(next.display);
      }
      return { date: row.date || "", account: row.account || "", units: postedUnits(row), flag: row.flag || "", running };
    });
    app.innerHTML = `${head}${chartHtml}<div id="account-journal"></div>`;
    mountChartData(app.querySelector(".report-chart"), chart);
    wireCharts(app);
    wireChartWindow();
    mountRawTable(document.getElementById("account-journal"), { columns: ["date", "account", "flag", "units", "running"], rows: detailRows }, { runningLabel: t("runningBalance") });
  } catch (error) { app.innerHTML = renderError(error); }
}
function postedAmountValue(row) {
  const units = row.units;
  if (units && typeof units === "object" && units.exact != null) return { exact: units.exact, display: units.display != null ? units.display : units.exact };
  if (units != null) return { exact: String(units), display: String(units) };
  return null;
}
function addExact(left, right) {
  const a = decimalParts(left);
  const b = decimalParts(right);
  if (!a || !b) return { exact: String(left), display: String(left) };
  const numerator = a.numerator * b.denominator + b.numerator * a.denominator;
  const denominator = a.denominator * b.denominator;
  if (numerator === 0n) return { exact: "0", display: "0" };
  // Reduce by the GCD so the result is exact and minimal.
  let n = numerator < 0n ? -numerator : numerator;
  let d = denominator;
  while (d !== 0n) { const r = n % d; n = d; d = r; }
  const gcd = n;
  const reducedNum = numerator / gcd;
  const reducedDen = denominator / gcd;
  if (reducedDen === 1n) return { exact: String(reducedNum), display: String(reducedNum) };
  // A denominator with only 2 and 5 prime factors terminates as a decimal;
  // anything else stays a reduced fraction.
  let remaining = reducedDen;
  while (remaining % 2n === 0n) remaining /= 2n;
  while (remaining % 5n === 0n) remaining /= 5n;
  if (remaining === 1n) {
    const sign = reducedNum < 0n ? "-" : "";
    const absNum = reducedNum < 0n ? -reducedNum : reducedNum;
    const integer = absNum / reducedDen;
    const remainder = absNum % reducedDen;
    if (remainder === 0n) return { exact: `${reducedNum}/${reducedDen}`, display: `${sign}${integer}` };
    // Scale the remainder to the full decimal expansion of the denominator.
    let scale = 1n;
    let digits = "";
    let r = remainder;
    while (r !== 0n) {
      r *= 10n;
      digits += String(r / reducedDen);
      r %= reducedDen;
      scale *= 10n;
    }
    return { exact: `${reducedNum}/${reducedDen}`, display: `${sign}${integer}.${digits}` };
  }
  return { exact: `${reducedNum}/${reducedDen}`, display: `${reducedNum}/${reducedDen}` };
}
function mountRawTable(container, result, options = {}) {
  const columns = result.columns.map((column) => column === "running" && options.runningLabel ? options.runningLabel : column);
  container.innerHTML = `<div class="table-wrap"><table><thead><tr>${columns.map((column) => `<th>${escapeHTML(column)}</th>`).join("")}</tr></thead><tbody>${result.rows.map((row) => `<tr>${result.columns.map((column) => `<td>${escapeHTML(row[column] == null ? "" : String(row[column]))}</td>`).join("")}</tr>`).join("")}</tbody></table></div>`;
}
function diagnosticPhase(code) {
  if (code.startsWith("E-INCLUDE") || code.startsWith("E-SOURCE")) return "fix-first-source";
  if (code.startsWith("E-PARSE")) return "fix-first-syntax";
  return "recheck-after-semantic";
}
function diagnosticRow(value) {
  const span = value && value.span ? value.span : {};
  return {
    code: String(value?.code || ""), severity: String(value?.severity || "error"), path: String(value?.path || ""),
    line: Number(value?.line || span.start_line || 0), column: Number(value?.column || span.start_column || 0), message: String(value?.message || ""),
  };
}
function renderRepairGuide(guide, row) {
  const example = guide.example || {};
  const contextKey = `${row.path}:${row.line}`;
  return `<div class="repair-guide compact"><p class="repair-action"><strong>${escapeHTML(guide.short_action || "")}</strong></p>${row.path ? `<p><a href="/source?path=${encodeURIComponent(row.path)}">${escapeHTML(`${row.path}:${row.line}:${row.column}`)}</a></p>` : ""}<h4>${escapeHTML(t("whatHappened"))}</h4><p>${escapeHTML(guide.what || "")}</p><h4>${escapeHTML(t("whyBlocks"))}</h4><p>${escapeHTML(guide.why || "")}</p><h4>${escapeHTML(t("whereToInspect"))}</h4><p>${escapeHTML((guide.inspect || []).join(" "))}</p><h4>${escapeHTML(t("safeChecks"))}</h4><p>${escapeHTML((guide.safe_steps || []).join(" "))}</p><h4>${escapeHTML(t("genericExample"))}</h4><div class="example-grid"><pre>${escapeHTML(example.before || "")}</pre><pre>${escapeHTML(example.after || "")}</pre></div><p class="muted">${escapeHTML(example.note || "")}</p><h4>${escapeHTML(t("nextStep"))}</h4><p>${escapeHTML(guide.revalidate || "")}</p><p><a href="/help/${encodeURIComponent(guide.topic || `diagnostics/${row.code}`)}">${escapeHTML(t("helpTopic"))}</a></p>${row.path && row.line ? `<button type="button" data-context-path="${escapeHTML(row.path)}" data-context-line="${row.line}">${escapeHTML(t("showLocalContext"))}</button><pre class="source-content context-snippet" data-context-output="${escapeHTML(contextKey)}" hidden></pre>` : ""}</div>`;
}
async function loadLegacyGuide(details, row) {
  const body = details.querySelector("[data-guidance-body]");
  if (!body || body.dataset.loaded === "true") return;
  body.dataset.loaded = "true";
  body.innerHTML = `<p class="muted">${escapeHTML(t("loadingGuide"))}</p>`;
  try {
    const guide = await api(`/api/v1/help?topic=${encodeURIComponent(`diagnostics/${row.code}`)}&locale=${encodeURIComponent(locale)}`);
    body.innerHTML = renderRepairGuide(guide, row);
    body.querySelector("[data-context-path]")?.addEventListener("click", async (event) => {
      const button = event.currentTarget;
      const path = button.dataset.contextPath || "";
      const line = Number(button.dataset.contextLine || 0);
      const output = [...body.querySelectorAll("[data-context-output]")].find((candidate) => candidate.dataset.contextOutput === `${path}:${line}`);
      button.disabled = true;
      try {
        const context = await api(`/api/v1/diagnostics/context?path=${encodeURIComponent(path)}&line=${line}`);
        if (!output) return;
        if (context.available) {
          output.textContent = (context.lines || []).map((item) => `${item.line}: ${item.content}`).join("\n");
          output.hidden = false;
        } else {
          output.textContent = context.reason || t("contextUnavailable");
          output.hidden = false;
        }
      } catch (error) {
        if (output) {
          output.textContent = error.message || t("contextUnavailable");
          output.hidden = false;
        }
      } finally { button.disabled = false; }
    });
  } catch (error) { body.innerHTML = `<p class="error">${escapeHTML(error.message || t("noGuidance"))}</p>`; }
}
function renderDiagnosticGroup(title, rows) {
  if (!rows.length) return "";
  return `<section class="diagnostic-group"><h3>${escapeHTML(title)}</h3>${rows.map((row) => row.severity === "error"
    ? `<details class="diagnostic-card" data-diagnostic-code="${escapeHTML(row.code)}" data-diagnostic-path="${escapeHTML(row.path)}" data-diagnostic-line="${row.line}"><summary><code>${escapeHTML(row.code)}</code> <span class="muted">${escapeHTML(row.path)}:${row.line}:${row.column}</span><p>${escapeHTML(row.message)}</p></summary><div data-guidance-body></div></details>`
    : `<article class="diagnostic-card warning"><header><code>${escapeHTML(row.code)}</code> <span class="muted">${escapeHTML(row.path)}:${row.line}:${row.column}</span><p>${escapeHTML(row.message)}</p></header></article>`).join("")}</section>`;
}
async function renderDiagnostics() {
  app.innerHTML = `<p class="muted">${escapeHTML(t("loading"))}</p>`;
  try {
    const values = await api(`/api/v1/diagnostics?locale=${encodeURIComponent(locale)}`);
    const rows = Array.isArray(values) ? values.map(diagnosticRow) : [];
    const first = rows.filter((row) => diagnosticPhase(row.code) !== "recheck-after-semantic");
    const later = rows.filter((row) => diagnosticPhase(row.code) === "recheck-after-semantic");
    const groups = rows.length ? `${renderDiagnosticGroup(t("fixFirst"), first)}${renderDiagnosticGroup(t("recheckAfter"), later)}` : `<p>${escapeHTML(t("noErrors"))}</p>`;
    app.innerHTML = `<div class="headerline"><h2>${escapeHTML(t("diagnostics"))}</h2></div><p class="muted">${escapeHTML(t("repairOrderHint"))}</p>${groups}`;
    app.querySelectorAll("details.diagnostic-card").forEach((details) => {
      const row = [...first, ...later].find((candidate) => candidate.severity === "error" && candidate.code === details.dataset.diagnosticCode && candidate.path === details.dataset.diagnosticPath && candidate.line === Number(details.dataset.diagnosticLine));
      if (!row) return;
      details.addEventListener("toggle", () => { if (details.open) loadLegacyGuide(details, row); });
    });
  } catch (error) { app.innerHTML = renderError(error); }
}
async function renderSource() {
  app.innerHTML = `<p class="muted">${escapeHTML(t("loading"))}</p>`;
  try {
    const listing = await api("/api/v1/source");
    const paths = Array.isArray(listing.paths) ? listing.paths : [];
    if (!paths.length) { app.innerHTML = `<p class="muted">${escapeHTML(t("empty"))}</p>`; return; }
    app.innerHTML = `<p class="muted">${escapeHTML(t("sourceHint"))}</p><div class="toolbar">${paths.map((path) => `<button type="button" data-source-path="${escapeHTML(path)}">${escapeHTML(path)}</button>`).join("")}</div><h3 id="source-file">${escapeHTML(t("file"))}</h3><pre id="source-content" class="source-content">${escapeHTML(t("empty"))}</pre>`;
    const loadSource = async (path) => {
      const fileHeading = document.getElementById("source-file");
      const content = document.getElementById("source-content");
      fileHeading.textContent = path;
      content.textContent = t("loading");
      try { const file = await api(`/api/v1/source?path=${encodeURIComponent(path)}`); content.textContent = file.content || ""; }
      catch (error) { content.textContent = error.message || String(error); }
    };
    app.querySelectorAll("button[data-source-path]").forEach((button) => button.addEventListener("click", () => loadSource(button.dataset.sourcePath || "")));
    const requested = params.get("path");
    if (requested && paths.includes(requested)) loadSource(requested);
  } catch (error) { app.innerHTML = renderError(error); }
}
function highlightSource(text) {
  const escaped = escapeHTML(text);
  return escaped
    .replace(/(^|\n)(\s*;[^\n]*)/g, '$1<span class="syntax-comment">$2</span>')
    .replace(/\b(\d{4}-\d{2}-\d{2})\b/g, '<span class="syntax-date">$1</span>')
    .replace(/\b(open|close|commodity|balance|pad|event|price|document|note|custom|include|option|plugin)\b/g, '<span class="syntax-keyword">$1</span>')
    .replace(/(#[A-Za-z0-9_-]+|\^[A-Za-z0-9_-]+)/g, '<span class="syntax-tag">$1</span>')
    .replace(/(&quot;[^&]*?&quot;)/g, '<span class="syntax-string">$1</span>');
}
function renderDiagnosticResult(values) {
  const rows = Array.isArray(values) ? values.map((value) => ({ severity: value.severity, code: value.code, path: value.path, line: value.line, column: value.column, message: value.message })) : [];
  return { columns: ["severity", "code", "path", "line", "column", "message"], rows };
}
async function renderEditor() {
  app.innerHTML = `<p class="muted">${escapeHTML(t("loading"))}</p>`;
  try {
    const listing = await api("/api/v1/editor");
    const paths = Array.isArray(listing.paths) ? listing.paths : [];
    if (!paths.length) { app.innerHTML = `<p class="muted">${escapeHTML(t("noFile"))}</p>`; return; }
    app.innerHTML = `<div class="editor-layout"><aside class="editor-files"><h3>${escapeHTML(t("files"))}</h3><select id="editor-file" size="${Math.min(Math.max(paths.length, 2), 12)}">${paths.map((path) => `<option value="${escapeHTML(path)}">${escapeHTML(path)}</option>`).join("")}</select></aside><section class="editor-pane"><div class="toolbar"><button id="editor-validate" type="button">${escapeHTML(t("validate"))}</button><button id="editor-save" type="button">${escapeHTML(t("save"))}</button><span id="editor-status" class="muted"></span></div><label class="sr-only" for="editor-buffer">${escapeHTML(t("editor"))}</label><div class="editor-code"><pre id="editor-lines" class="line-numbers" aria-hidden="true">1</pre><textarea id="editor-buffer" spellcheck="false"></textarea></div><h3>${escapeHTML(t("syntax"))}</h3><pre id="editor-highlight" class="source-content syntax-preview"></pre><div id="editor-diagnostics"></div></section></div>`;
    const picker = document.getElementById("editor-file");
    const buffer = document.getElementById("editor-buffer");
    const lines = document.getElementById("editor-lines");
    const highlight = document.getElementById("editor-highlight");
    const status = document.getElementById("editor-status");
    const diagnostics = document.getElementById("editor-diagnostics");
    let snapshotID = listing.snapshot_id || "";
    picker.value = paths[0];
    const updateHighlight = () => {
      highlight.innerHTML = highlightSource(buffer.value);
      lines.textContent = Array.from({ length: Math.max(1, buffer.value.split("\n").length) }, (_, index) => index + 1).join("\n");
    };
    // Debounce re-highlighting so typing stays fluid; the preview updates a
    // few hundred milliseconds after the user stops typing.
    let highlightTimer = null;
    buffer.addEventListener("input", () => {
      clearTimeout(highlightTimer);
      highlightTimer = setTimeout(updateHighlight, 200);
    });
    buffer.addEventListener("scroll", () => { lines.scrollTop = buffer.scrollTop; highlight.scrollTop = buffer.scrollTop; });
    const load = async (path) => {
      status.textContent = t("loading");
      try {
        const file = await api(`/api/v1/editor/file?path=${encodeURIComponent(path)}`);
        buffer.value = file.content || "";
        snapshotID = file.snapshot_id || snapshotID;
        updateHighlight();
        status.textContent = file.path || path;
      } catch (error) { status.textContent = error.message || String(error); }
    };
    picker.addEventListener("change", () => load(picker.value));
    const showDiagnostics = (values) => { diagnostics.innerHTML = ""; if (Array.isArray(values) && values.length) mountTable(diagnostics, renderDiagnosticResult(values)); };
    document.getElementById("editor-validate").addEventListener("click", async () => {
      try {
        const result = await apiJSON("/api/v1/editor/validate", "POST", { path: picker.value, content: buffer.value, expected_snapshot_id: snapshotID });
        status.textContent = result.valid ? t("valid") : t("previewDiagnostics");
        showDiagnostics(result.diagnostics);
      } catch (error) { status.textContent = error.message || String(error); }
    });
    document.getElementById("editor-save").addEventListener("click", async () => {
      try {
        const result = await apiJSON("/api/v1/editor/save", "POST", { path: picker.value, content: buffer.value, expected_snapshot_id: snapshotID });
        snapshotID = result.snapshot_id || snapshotID;
        status.textContent = `${t("save")}: ${t("valid")}${result.backup ? ` · ${t("backup")}: ${result.backup}` : ""}`;
        showDiagnostics(result.diagnostics);
      } catch (error) { status.textContent = error.message || String(error); }
    });
    buffer.addEventListener("keydown", (event) => {
      if (!(event.ctrlKey || event.metaKey)) return;
      if (event.key.toLowerCase() === "s") { event.preventDefault(); document.getElementById("editor-save").click(); }
      if (event.key === "Enter") { event.preventDefault(); document.getElementById("editor-validate").click(); }
    });
    await load(paths[0]);
  } catch (error) { app.innerHTML = renderError(error); }
}
async function renderImport() {
  app.innerHTML = `<p class="muted">${escapeHTML(t("loading"))}</p>`;
  try {
    const [targets, adaptersResponse] = await Promise.all([api("/api/v1/import/targets"), api("/api/v1/import/adapters")]);
    const paths = Array.isArray(targets.paths) ? targets.paths : [];
    app.innerHTML = `<p class="muted">${escapeHTML(t("documentsHint"))}</p><div class="toolbar"><label>${escapeHTML(t("chooseFile"))} <input id="import-file" type="file" accept=".bean,.beancount,text/plain"></label><label>${escapeHTML(t("files"))} <input id="import-path" type="text" value="import.bean" pattern="[^/\\\\]+\\.(bean|beancount)"></label><label>${escapeHTML(t("target"))} <select id="import-target">${paths.map((path) => `<option value="${escapeHTML(path)}">${escapeHTML(path)}</option>`).join("")}</select></label></div><textarea id="import-buffer" spellcheck="false" placeholder="${escapeHTML(t("chooseFile"))}"></textarea><div class="toolbar"><button id="import-preview" type="button">${escapeHTML(t("preview"))}</button><button id="import-commit" type="button" disabled>${escapeHTML(t("commit"))}</button><span id="import-status" class="muted"></span></div><p id="import-diff" class="muted"></p><div id="import-result"></div>`;
    const adapters = Array.isArray(adaptersResponse.adapters) ? adaptersResponse.adapters : [];
    const importToolbar = document.querySelector(".toolbar");
    if (importToolbar) importToolbar.insertAdjacentHTML("afterbegin", `<label>${escapeHTML(t("adapter"))} <select id="import-adapter">${adapters.map((adapter) => `<option value="${escapeHTML(adapter.id)}">${escapeHTML(adapter.label)}</option>`).join("")}</select></label><label>${escapeHTML(t("offset"))} <input id="import-offset" type="text" value="Equity:Opening"></label><label>${escapeHTML(t("currency"))} <input id="import-currency" type="text" value="USD" maxlength="12"></label>`);
    const filePicker = document.getElementById("import-file");
    const adapterPicker = document.getElementById("import-adapter");
    const pathInput = document.getElementById("import-path");
    const buffer = document.getElementById("import-buffer");
    const status = document.getElementById("import-status");
    const output = document.getElementById("import-result");
    const commit = document.getElementById("import-commit");
    let previewID = "";
    let snapshotID = targets.snapshot_id || "";
    if (adapterPicker) adapterPicker.addEventListener("change", () => {
      if (adapterPicker.value === "csv" && /\.(bean|beancount)$/i.test(pathInput.value)) pathInput.value = pathInput.value.replace(/\.(bean|beancount)$/i, ".csv");
      if (adapterPicker.value === "beancount" && /\.csv$/i.test(pathInput.value)) pathInput.value = pathInput.value.replace(/\.csv$/i, ".bean");
    });
    filePicker.addEventListener("change", () => {
      const file = filePicker.files && filePicker.files[0];
      if (!file) return;
      pathInput.value = file.name;
      const reader = new FileReader();
      reader.onload = () => { buffer.value = typeof reader.result === "string" ? reader.result : ""; };
      reader.readAsText(file);
    });
    document.getElementById("import-preview").addEventListener("click", async () => {
      try {
        const result = await apiJSON("/api/v1/import/preview", "POST", { path: pathInput.value, content: buffer.value, adapter: adapterPicker?.value || "beancount", mapping: { offset_account: document.getElementById("import-offset")?.value || "", currency: document.getElementById("import-currency")?.value || "" } });
        previewID = result.preview_id || "";
        snapshotID = targets.snapshot_id || snapshotID;
        commit.disabled = !previewID;
        status.textContent = result.valid ? t("valid") : t("requestFailed");
        document.getElementById("import-diff").textContent = result.diff ? `${result.diff.added_lines} lines · ${result.diff.bytes} bytes` : "";
        output.innerHTML = "";
        if (result.diagnostics && result.diagnostics.length) { const box = document.createElement("div"); output.appendChild(box); mountTable(box, renderDiagnosticResult(result.diagnostics)); }
        if (result.rows && result.rows.rows) { const table = document.createElement("div"); output.appendChild(table); mountTable(table, result.rows); }
      } catch (error) { status.textContent = error.message || String(error); }
    });
    commit.addEventListener("click", async () => {
      if (!previewID) return;
      try {
        const result = await apiJSON("/api/v1/import/commit", "POST", { preview_id: previewID, target: document.getElementById("import-target").value, expected_snapshot_id: snapshotID });
        status.textContent = result.published ? `${t("commit")}: ${t("valid")}` : t("requestFailed");
        commit.disabled = true;
        previewID = "";
      } catch (error) { status.textContent = error.message || String(error); }
    });
  } catch (error) { app.innerHTML = renderError(error); }
}
async function renderOptions() {
  app.innerHTML = `<p class="muted">${escapeHTML(t("loading"))}</p>`;
  try {
    const response = await api("/api/v1/options");
    const values = response.options || {};
    const ledgerEntries = Object.entries(values).filter(([key]) => !["locale", "currency", "time"].includes(key));
    const readonlyTable = ledgerEntries.length ? `<div class="options-readonly"><h3>${escapeHTML(t("optionsFromLedger"))}</h3><div class="table-wrap"><table><thead><tr><th>option</th><th>value</th></tr></thead><tbody>${ledgerEntries.map(([key, value]) => `<tr><td>${escapeHTML(key)}</td><td>${escapeHTML(String(value))}</td></tr>`).join("")}</tbody></table></div></div>` : "";
    const themeSelect = `<div class="toolbar options-form"><label>${escapeHTML(t("theme"))} <select id="options-theme"><option value="dark">${escapeHTML(t("dark"))}</option><option value="light">${escapeHTML(t("light"))}</option><option value="system">${escapeHTML(t("system"))}</option></select></label></div>`;
    app.innerHTML = `<div class="toolbar options-form"><label>${escapeHTML(t("language"))} <select id="options-locale"><option value="en">English</option><option value="zh-CN">简体中文</option></select></label><label>${escapeHTML(t("currency"))} <input id="options-currency" value="${escapeHTML(values.currency || globalState.currency)}" maxlength="12"></label><label>${escapeHTML(t("time"))} <select id="options-time"><option value="all">${escapeHTML(t("allPeriods"))}</option><option value="year">${escapeHTML(t("yearly"))}</option><option value="month">${escapeHTML(t("monthly"))}</option></select></label><button id="options-save" type="button">${escapeHTML(t("save"))}</button><span id="options-status" class="muted"></span></div>${themeSelect}${readonlyTable}<p class="muted">${escapeHTML(t("subtitle"))}</p>`;
    const localeInput = document.getElementById("options-locale");
    const currencyInput = document.getElementById("options-currency");
    const timeInput = document.getElementById("options-time");
    const themeInput = document.getElementById("options-theme");
    localeInput.value = values.locale || locale;
    timeInput.value = values.time || globalState.time;
    themeInput.value = theme === "system" ? "system" : theme;
    themeInput.addEventListener("change", () => { applyTheme(themeInput.value); render(); });
    document.getElementById("options-save").addEventListener("click", async () => {
      const next = { locale: localeInput.value, currency: currencyInput.value.trim().toUpperCase(), time: timeInput.value };
      try {
        await apiJSON("/api/v1/options", "POST", next);
        locale = next.locale === "zh-CN" ? "zh-CN" : "en";
        localStorage.setItem("orangecount-locale", locale);
        globalState.currency = next.currency || "";
        globalState.time = next.time || "all";
        updateURL();
        document.getElementById("options-status").textContent = t("saved");
        render();
      } catch (error) { document.getElementById("options-status").textContent = error.message || String(error); }
    });
  } catch (error) { app.innerHTML = renderError(error); }
}
async function renderHelp() {
  app.innerHTML = `<p class="muted">${escapeHTML(t("loading"))}</p>`;
  try {
    if (helpPage.startsWith("diagnostics/")) {
      const guide = await api(`/api/v1/help?topic=${encodeURIComponent(helpPage)}&locale=${encodeURIComponent(locale)}`);
      app.innerHTML = `<div class="headerline"><h2>${escapeHTML(guide.code || helpPage)}</h2></div><p><a href="/help">‹ ${escapeHTML(t("help"))}</a></p>${renderRepairGuide(guide, { code: guide.code || helpPage.slice("diagnostics/".length), path: "", line: 0 })}`;
      return;
    }
    const response = await api(`/api/v1/help?locale=${encodeURIComponent(locale)}`);
    const sections = Array.isArray(response.sections) ? response.sections : [];
    app.innerHTML = `<input id="help-search" type="search" placeholder="${escapeHTML(t("searchHelp"))}" aria-label="${escapeHTML(t("searchHelp"))}"><div id="help-sections" class="help-sections"></div>`;
    const search = document.getElementById("help-search");
    const render = () => {
      const needle = search.value.trim().toLowerCase();
      document.getElementById("help-sections").innerHTML = sections.filter((section) => !needle || `${section.title} ${section.body}`.toLowerCase().includes(needle)).map((section) => `<article class="card"><h3>${escapeHTML(section.title)}</h3><p>${escapeHTML(section.body)}</p></article>`).join("") || `<p class="muted">${escapeHTML(t("empty"))}</p>`;
    };
    search.addEventListener("input", render);
    render();
  } catch (error) { app.innerHTML = renderError(error); }
}
// renderPivot is the Excel-style report builder: pick rows (time bucket),
// columns (currency or account level), and a value (period totals or ending
// balance), then render the server's cross-tab. No query syntax needed.
function renderPivot() {
  const state = pivotState();
  const options = (values, active) => values.map(([value, label]) => `<option value="${value}" ${value === active ? "selected" : ""}>${escapeHTML(label)}</option>`).join("");
  app.innerHTML = `<div class="toolbar report-toolbar" role="group" aria-label="${escapeHTML(t("pivot"))}">
    <label>${escapeHTML(t("pivotRows"))} <select id="pivot-rows">${options([["month", t("pivotMonth")], ["quarter", t("pivotQuarter")], ["year", t("pivotYear")]], state.rows)}</select></label>
    <label>${escapeHTML(t("pivotColumns"))} <select id="pivot-columns">${options([["", t("pivotNone")], ["root1", t("pivotRoot1")], ["root2", t("pivotRoot2")], ["root3", t("pivotRoot3")]], state.columns)}</select></label>
    <label>${escapeHTML(t("pivotValues"))} <select id="pivot-values">${options([["sum", t("pivotSum")], ["balance", t("pivotBalance")]], state.values)}</select></label>
    <label>${escapeHTML(t("pivotAccount"))} <input id="pivot-account" type="text" value="${escapeHTML(state.account)}" placeholder="Expenses"></label>
    <button id="pivot-apply" type="button">${escapeHTML(t("pivotApply"))}</button>
    <a class="button" id="pivot-export" href="${escapeHTML(pivotURL(state, "csv"))}" download>${escapeHTML(t("exportCSV"))}</a>
  </div><div id="pivot-result"></div>`;
  const run = async () => {
    const next = {
      rows: document.getElementById("pivot-rows").value,
      columns: document.getElementById("pivot-columns").value,
      values: document.getElementById("pivot-values").value,
      account: document.getElementById("pivot-account").value.trim(),
    };
    localStorage.setItem("orangecount-pivot", JSON.stringify(next));
    document.getElementById("pivot-export").href = pivotURL(next, "csv");
    const output = document.getElementById("pivot-result");
    output.innerHTML = `<p class="muted">${escapeHTML(t("loading"))}</p>`;
    try { mountTable(output, await api(pivotURL(next))); }
    catch (error) { output.innerHTML = renderError(error); }
  };
  document.getElementById("pivot-apply").addEventListener("click", run);
  ["pivot-rows", "pivot-columns", "pivot-values"].forEach((id) => document.getElementById(id).addEventListener("change", run));
  run();
}

// pivotState loads the remembered pivot selections.
function pivotState() {
  try { return { rows: "month", columns: "", values: "sum", account: "", ...JSON.parse(localStorage.getItem("orangecount-pivot") || "{}") }; }
  catch { return { rows: "month", columns: "", values: "sum", account: "" }; }
}

// pivotURL builds the pivot report request, carrying the global filters so
// the pivot respects the same filtered view as the other pages.
function pivotURL(state, format) {
  const search = globalQuery();
  search.set("rows", state.rows);
  search.set("columns", state.columns);
  search.set("values", state.values);
  if (state.account) search.set("account", state.account);
  if (format) search.set("format", format);
  return `/api/v1/reports/pivot?${search.toString()}`;
}

function renderQuery() {
  const saved = JSON.parse(localStorage.getItem("orangecount-saved-queries") || "[]");
  app.innerHTML = `<textarea id="query-text" aria-label="${escapeHTML(t("query"))}">${escapeHTML(params.get("q") || t("queryHint"))}</textarea><div class="toolbar"><button id="run-query" type="button">${escapeHTML(t("run"))}</button><input id="query-name" type="text" placeholder="${escapeHTML(t("queryName"))}"><button id="save-query" type="button">${escapeHTML(t("save"))}</button><label>${escapeHTML(t("saved"))} <select id="saved-queries"><option value="">—</option>${saved.map((entry) => `<option value="${escapeHTML(entry.name)}">${escapeHTML(entry.name)}</option>`).join("")}</select></label><a class="button" id="query-csv" href="#" download>${escapeHTML(t("exportCSV"))}</a></div><div id="query-result"></div>`;
  const queryText = document.getElementById("query-text");
  const csvLink = document.getElementById("query-csv");
  const updateCSV = () => { csvLink.href = `/api/v1/query?format=csv&q=${encodeURIComponent(queryText.value)}`; };
  updateCSV();
  queryText.addEventListener("input", updateCSV);
  document.getElementById("saved-queries").addEventListener("change", (event) => {
    const selected = saved.find((entry) => entry.name === event.target.value);
    if (selected) { queryText.value = selected.query; updateCSV(); }
  });
  document.getElementById("save-query").addEventListener("click", () => {
    const name = document.getElementById("query-name").value.trim();
    if (!name) return;
    const next = saved.filter((entry) => entry.name !== name).concat({ name, query: queryText.value });
    localStorage.setItem("orangecount-saved-queries", JSON.stringify(next));
    document.getElementById("query-name").value = "";
  });
  document.getElementById("run-query").addEventListener("click", async () => {
    const query = queryText.value;
    const next = new URL(window.location.href);
    next.searchParams.set("q", query);
    window.history.replaceState({}, "", next);
    const output = document.getElementById("query-result");
    const previous = output.innerHTML;
    output.innerHTML = `<p class="muted">${escapeHTML(t("loading"))}</p>`;
    try { mountQueryResult(output, await api(`/api/v1/query?q=${encodeURIComponent(query)}`)); }
    catch (error) { output.innerHTML = `${renderError(error)}${previous}`; }
  });
}

// mountQueryResult renders the workbench result with a flat/cross-tab toggle:
// two columns pivot into rows+values, three or more pivot on the first two
// dimension columns with the final column as the value.
function mountQueryResult(container, result) {
  const columns = result.columns || [];
  const pivotable = columns.length >= 2;
  if (!pivotable) { mountTable(container, result); return; }
  container.innerHTML = `<div class="toolbar"><button id="query-pivot-toggle" type="button" aria-pressed="false">${escapeHTML(t("pivotCross"))}</button></div><div id="query-pivot-body"></div>`;
  const body = container.querySelector("#query-pivot-body");
  let crossed = false;
  const render = () => {
    const button = container.querySelector("#query-pivot-toggle");
    button.setAttribute("aria-pressed", crossed ? "true" : "false");
    button.textContent = t(crossed ? "pivotFlat" : "pivotCross");
    if (!crossed) { mountTable(body, result); return; }
    const cross = crossTab(result);
    if (!cross) { body.innerHTML = `<p class="muted">${escapeHTML(t("pivotNeedsShape"))}</p>`; return; }
    body.innerHTML = cross;
    wireTables(body, {});
  };
  container.querySelector("#query-pivot-toggle").addEventListener("click", () => { crossed = !crossed; render(); });
  render();
}

// crossTab pivots a flat result client-side: the first column becomes rows,
// the second becomes columns (when distinct), and the last column supplies
// the values. Returns HTML or null when the shape does not fit.
function crossTab(result) {
  const columns = result.columns || [];
  const rows = result.rows || [];
  if (columns.length < 2 || !rows.length) return null;
  const rowKey = columns[0];
  const valueKey = columns[columns.length - 1];
  const columnKey = columns.length >= 3 ? columns[1] : null;
  const order = [];
  const columnLabels = [];
  const seenColumns = new Set();
  const cells = new Map();
  for (const row of rows) {
    const rowLabel = String(row[rowKey] ?? "");
    const columnLabel = columnKey ? String(row[columnKey] ?? "") : t("pivotSum");
    if (!seenColumns.has(columnLabel)) { seenColumns.add(columnLabel); columnLabels.push(columnLabel); }
    if (!cells.has(rowLabel)) { cells.set(rowLabel, new Map()); order.push(rowLabel); }
    cells.get(rowLabel).set(columnLabel, row[valueKey]);
  }
  const head = `<thead><tr><th>${escapeHTML(rowKey)}</th>${columnLabels.map((label) => `<th>${escapeHTML(label)}</th>`).join("")}</tr></thead>`;
  const body = order.map((rowLabel) => `<tr><th>${escapeHTML(rowLabel)}</th>${columnLabels.map((label) => {
    const value = cells.get(rowLabel).get(label);
    const display = value && typeof value === "object" && "display" in value ? value.display : (value ?? "");
    return `<td>${escapeHTML(String(display))}</td>`;
  }).join("")}</tr>`).join("");
  return `<div class="table-wrap"><table class="report-table">${head}<tbody>${body}</tbody></table></div>`;
}
function render() {
  document.documentElement.lang = locale;
  pageTitle.textContent = t(navRoutes.find(([route]) => route === view)?.[1] || "overview");
  subtitle.textContent = t("subtitle");
  localePicker.value = locale;
  timePicker.value = globalState.time;
  accountPicker.value = globalState.account;
  globalFilter.value = globalState.filter;
  currencySwitch.querySelectorAll("button[data-currency]").forEach((button) => button.setAttribute("aria-pressed", button.dataset.currency === globalState.currency ? "true" : "false"));
  renderBrand();
  renderNavigation();
  if (view === "overview") return renderOverview();
  if (view === "source") return renderSource();
  if (view === "diagnostics") return renderDiagnostics();
  if (view === "query") return renderQuery();
  if (view === "pivot") return renderPivot();
  if (view === "editor") return renderEditor();
  if (view === "import") return renderImport();
  if (view === "options") return renderOptions();
  if (view === "help") return renderHelp();
  return renderReport(view);
}
// renderBrand builds the topbar breadcrumb "‹ledger title› › current page" from
// the ledger option title, falling back to the product name when no title is
// set. Keeps the brand link pointing at the home page.
function renderBrand() {
  const pageLabel = t(navRoutes.find(([route]) => route === view)?.[1] || "overview");
  const title = ledgerTitle || "OrangeCount";
  brand.innerHTML = `${escapeHTML(title)}${view && view !== "overview" ? `<span class="brand-sep">›</span><span class="brand-page">${escapeHTML(pageLabel)}</span>` : ""}`;
  brand.setAttribute("aria-label", title);
}
// applyTheme applies the System/Dark/Light theme to the document root and
// persists the choice. System resolves to the OS color-scheme preference.
function applyTheme(value) {
  theme = value === "system" ? "system" : (value === "light" ? "light" : "dark");
  localStorage.setItem("orangecount-theme", theme);
  if (theme === "system") {
    document.documentElement.removeAttribute("data-theme");
  } else {
    document.documentElement.setAttribute("data-theme", theme);
  }
}
async function loadAccountOptions() {
  try {
    const result = await api("/api/v1/query?q=SELECT%20account%20FROM%20accounts%20ORDER%20BY%20account");
    const values = Array.isArray(result.rows) ? result.rows.map((row) => row.account).filter((value) => typeof value === "string") : [];
    const unique = [...new Set(values)];
    accountPicker.innerHTML = `<option value="">All accounts</option>${unique.map((value) => `<option value="${escapeHTML(value)}">${escapeHTML(value)}</option>`).join("")}`;
    accountPicker.value = globalState.account;
  } catch (_) { /* The shell remains usable when account suggestions are unavailable. */ }
}
menuToggle.addEventListener("click", () => {
  const open = sidebar.classList.toggle("open");
  menuToggle.setAttribute("aria-expanded", open ? "true" : "false");
});
timePicker.addEventListener("change", applyGlobalState);
accountPicker.addEventListener("change", applyGlobalState);
globalFilter.addEventListener("change", applyGlobalState);
globalFilter.addEventListener("keydown", (event) => { if (event.key === "Enter") applyGlobalState(); });
currencySwitch.addEventListener("click", (event) => {
  const button = event.target.closest("button[data-currency]");
  if (!button) return;
  globalState.currency = button.dataset.currency || "";
  updateURL();
  render();
});
localePicker.addEventListener("change", () => {
  locale = localePicker.value === "zh-CN" ? "zh-CN" : "en";
  localStorage.setItem("orangecount-locale", locale);
  const next = new URL(window.location.href);
  next.searchParams.set("locale", locale);
  window.history.replaceState({}, "", next);
  render();
});
async function bootstrap() {
  applyTheme(theme);
  try {
    const [status, options] = await Promise.all([api("/api/v1/status"), api("/api/v1/options")]);
    diagnosticCount = status.diagnostic_count || 0;
    const values = options.options || {};
    ledgerTitle = values.title || "";
    // operating_currency accumulates repeated declarations as a
    // space-separated list (see appendOperatingCurrency in the evaluator);
    // the first entry is the ledger's primary operating currency.
    const operatingCurrencies = (values.operating_currency || "").split(/\s+/).filter(Boolean);
    operatingCurrency = operatingCurrencies[0] || "";
    ledgerOptions = values;
    operatingCurrencies.forEach((currency) => {
      if (currencySwitch.querySelector(`button[data-currency="${CSS.escape(currency)}"]`)) return;
      const button = document.createElement("button");
      button.type = "button";
      button.dataset.currency = currency;
      button.textContent = currency;
      currencySwitch.appendChild(button);
    });
  } catch (_) { /* shell works without ledger metadata */ }
  render();
  loadAccountOptions();
}
bootstrap();
