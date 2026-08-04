export interface DecimalWire {
  display: string;
  exact: string;
  approximate: boolean;
}

export interface TreeNode {
  account: string;
  balance: Record<string, DecimalWire>;
  balance_children: Record<string, DecimalWire>;
  cost: Record<string, DecimalWire> | null;
  cost_children: Record<string, DecimalWire> | null;
  children: TreeNode[];
  has_txns: boolean;
}

export interface ChartPoint {
  date: string;
  value: DecimalWire;
}

export interface ChartSeries {
  label: string;
  points: ChartPoint[];
  stacked?: boolean;
}

export interface ReportChart {
  kind: string;
  title: string;
  unit: string;
  currency: string;
  valuation: string;
  period: string;
  interval: string;
  measure: string;
  availability?: string;
  series: ChartSeries[];
}

export interface TreeReport {
  date_range: { begin: string; end: string } | null;
  charts: ReportChart[];
  trees: TreeNode[];
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function decimal(value: unknown): value is DecimalWire {
  return isRecord(value) && typeof value.display === "string" && typeof value.exact === "string" && typeof value.approximate === "boolean";
}

function decimalMap(value: unknown): value is Record<string, DecimalWire> {
  return isRecord(value) && Object.values(value).every(decimal);
}

function treeNode(value: unknown): value is TreeNode {
  return isRecord(value) &&
    typeof value.account === "string" &&
    decimalMap(value.balance) &&
    decimalMap(value.balance_children) &&
    (value.cost === null || decimalMap(value.cost)) &&
    (value.cost_children === null || decimalMap(value.cost_children)) &&
    Array.isArray(value.children) && value.children.every(treeNode) &&
    typeof value.has_txns === "boolean";
}

function chart(value: unknown): value is ReportChart {
  if (!isRecord(value) || typeof value.kind !== "string" || typeof value.title !== "string" || typeof value.unit !== "string" || typeof value.currency !== "string" || typeof value.valuation !== "string" || typeof value.period !== "string" || typeof value.interval !== "string" || typeof value.measure !== "string" || !Array.isArray(value.series)) return false;
  return value.series.every((series) => {
    if (!isRecord(series) || typeof series.label !== "string" || !Array.isArray(series.points)) return false;
    return series.points.every((point) => isRecord(point) && typeof point.date === "string" && decimal(point.value));
  });
}

export function parseTreeReport(value: unknown): TreeReport {
  if (!isRecord(value) || !Array.isArray(value.trees) || !value.trees.every(treeNode) || !Array.isArray(value.charts) || !value.charts.every(chart)) {
    throw new Error("Adapter returned an invalid tree-report payload");
  }
  const dateRange = value.date_range;
  if (dateRange !== null && (!isRecord(dateRange) || typeof dateRange.begin !== "string" || typeof dateRange.end !== "string")) {
    throw new Error("Adapter returned an invalid report date range");
  }
  return {
    date_range: dateRange as TreeReport["date_range"],
    charts: value.charts as ReportChart[],
    trees: value.trees as TreeNode[],
  };
}

export function formatAmount(value: DecimalWire | undefined): string {
  return value?.display ?? "";
}

export function currenciesInTree(tree: TreeNode): string[] {
  const values = new Set<string>();
  function visit(node: TreeNode) {
    Object.keys(node.balance).forEach((currency) => values.add(currency));
    Object.keys(node.balance_children).forEach((currency) => values.add(currency));
    node.children.forEach(visit);
  }
  visit(tree);
  return [...values].sort();
}
