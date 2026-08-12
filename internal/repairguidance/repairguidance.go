// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package repairguidance contains static, localized explanations for blocking
// ledger diagnostics. It has no ledger, filesystem, or network dependency.
package repairguidance

import (
	"fmt"
	"sort"
	"strings"
)

const (
	LocaleEnglish = "en"
	LocaleChinese = "zh-CN"

	PhaseSource   = "fix-first-source"
	PhaseSyntax   = "fix-first-syntax"
	PhaseSemantic = "recheck-after-semantic"
)

// Example is a generic source example. It must never contain values copied
// from a user's ledger.
type Example struct {
	Before string `json:"before"`
	After  string `json:"after"`
	Note   string `json:"note"`
}

// Guide is the complete presentation-neutral anatomy of one repair topic.
type Guide struct {
	Code        string   `json:"code"`
	Topic       string   `json:"topic"`
	Phase       string   `json:"phase"`
	ShortAction string   `json:"short_action"`
	What        string   `json:"what"`
	Why         string   `json:"why"`
	Inspect     []string `json:"inspect"`
	SafeSteps   []string `json:"safe_steps"`
	Example     Example  `json:"example"`
	Revalidate  string   `json:"revalidate"`
}

type localizedGuide struct {
	English Guide
	Chinese Guide
}

func pair(code, phase, shortEn, shortZh, whatEn, whatZh, whyEn, whyZh, inspectEn, inspectZh, stepsEn, stepsZh, before, after, noteEn, noteZh, revalidateEn, revalidateZh string) localizedGuide {
	return localizedGuide{
		English: Guide{Code: code, Topic: "diagnostics/" + code, Phase: phase, ShortAction: shortEn, What: whatEn, Why: whyEn, Inspect: []string{inspectEn}, SafeSteps: []string{stepsEn}, Example: Example{Before: before, After: after, Note: noteEn}, Revalidate: revalidateEn},
		Chinese: Guide{Code: code, Topic: "diagnostics/" + code, Phase: phase, ShortAction: shortZh, What: whatZh, Why: whyZh, Inspect: []string{inspectZh}, SafeSteps: []string{stepsZh}, Example: Example{Before: before, After: after, Note: noteZh}, Revalidate: revalidateZh},
	}
}

var catalogue = map[string]localizedGuide{
	"E-INCLUDE-CYCLE": pair("E-INCLUDE-CYCLE", PhaseSource,
		"Break the include cycle", "解除 include 循环",
		"The include graph reaches a file it has already visited.", "include 图再次访问了已经加载过的文件。",
		"A cycle prevents the source graph from having one deterministic load order.", "循环会让源文件图没有唯一、确定的加载顺序。",
		"Trace each include path from the diagnostic and related location until the repeated file is found.", "从诊断和关联位置沿每条 include 路径追踪，直到找到重复文件。",
		"Remove or redirect one include edge while keeping the intended files in the graph.", "保留需要的文件，只移除或改向一条造成循环的 include 边。",
		"include \"parts.bean\"", "include \"other-parts.bean\"",
		"Choose one direction for the shared file; do not duplicate a file through a cycle.", "为共享文件选择单一方向；不要通过循环重复加载文件。",
		"Run orangecount check again and confirm the cycle diagnostic is gone.", "再次运行 orangecount check，确认循环诊断消失。"),
	"E-INCLUDE-READ": pair("E-INCLUDE-READ", PhaseSource,
		"Make the included file available", "使被 include 的文件可读取",
		"An include target could not be read.", "无法读取 include 目标文件。",
		"The ledger cannot be evaluated without every file in its include graph.", "缺少 include 图中的文件时，账本无法完成求值。",
		"Check the include path, spelling, and permissions relative to the including file.", "检查 include 路径、文件名拼写，以及相对于当前文件的权限。",
		"Correct the relative path or restore the intended file, then verify that it is readable.", "修正相对路径或恢复目标文件，然后确认它可读。",
		"include \"missing.bean\"", "include \"present.bean\"",
		"Use a path that exists beneath the source graph; do not broaden access just to silence the error.", "使用源文件图中确实存在的路径；不要为了消除错误而扩大访问权限。",
		"Run orangecount check again after the file is readable.", "文件可读后再次运行 orangecount check。"),
	"E-SOURCE-UTF8": pair("E-SOURCE-UTF8", PhaseSyntax,
		"Save the source as UTF-8", "将源文件保存为 UTF-8",
		"The source file contains bytes that are not valid UTF-8.", "源文件包含不是有效 UTF-8 的字节。",
		"The parser cannot reliably interpret text, account names, or directives in another encoding.", "解析器无法可靠解释其他编码中的文本、账户名或指令。",
		"Identify the affected file and inspect the editor or conversion tool encoding setting.", "确定受影响的文件，并检查编辑器或转换工具当前使用的编码。",
		"Convert or save a reviewed copy as UTF-8 without intentionally changing ledger text or line structure.", "在不有意改变账本文本或行结构的前提下，将副本转换或保存为 UTF-8。",
		"<bytes in a legacy encoding>", "<the same text encoded as UTF-8>",
		"Encoding conversion is a file operation; review the resulting diff before publishing it.", "编码转换属于文件操作；发布前请检查生成的差异。",
		"Run orangecount check again and inspect remaining parser diagnostics.", "再次运行 orangecount check，并处理剩余的解析诊断。"),
	"E-PARSE-DATE": pair("E-PARSE-DATE", PhaseSyntax,
		"Correct the date", "修正日期",
		"A date token is not a valid Beancount calendar date.", "日期标记不是有效的 Beancount 日历日期。",
		"Dates determine directive ordering and reporting periods, so an invalid date cannot be evaluated.", "日期决定指令顺序和报表期间，因此无效日期无法求值。",
		"Inspect the date token and check the YYYY-MM-DD shape and actual days in the month.", "查看日期标记，检查 YYYY-MM-DD 格式和该月份实际天数。",
		"Replace it with the intended valid calendar date; do not guess the business event date.", "替换为实际要表达的有效日期；不要猜测业务发生日期。",
		"2026-02-30 open Assets:Cash USD", "2026-02-28 open Assets:Cash USD",
		"The example chooses a calendar-valid date only; confirm the intended date from your records.", "示例只选择了日历上有效的日期；请根据记录确认真实日期。",
		"Run orangecount check again.", "再次运行 orangecount check。"),
	"E-PARSE-DIRECTIVE": pair("E-PARSE-DIRECTIVE", PhaseSyntax,
		"Correct the directive line", "修正指令行",
		"A top-level line is not a recognized or complete directive.", "顶层行不是可识别或完整的指令。",
		"Without a known directive shape, the parser cannot build the source entry needed for evaluation.", "无法识别指令形状时，解析器不能构建用于求值的源条目。",
		"Compare the line with the intended core directive and check whether indentation belongs to a parent.", "将该行与目标核心指令比较，并检查缩进是否属于父指令。",
		"Use the intended supported directive spelling and complete its required fields, or remove an accidental line after reviewing the diff.", "使用目标支持的指令拼写并补齐必填字段；误写行需检查差异后再删除。",
		"2026-01-01 opne Assets:Cash USD", "2026-01-01 open Assets:Cash USD",
		"The example fixes a spelling typo; it does not choose an accounting directive for you.", "示例只修正拼写错误，不替你选择会计指令。",
		"Run orangecount check and review continuation-line diagnostics.", "再次运行 orangecount check，并处理续行诊断。"),
	"E-PARSE-EXPECTED": pair("E-PARSE-EXPECTED", PhaseSyntax,
		"Add the expected token", "补齐期望的标记",
		"The parser found a token where the grammar requires another value or delimiter.", "解析器在语法需要另一个值或分隔符的位置遇到了标记。",
		"The incomplete structure makes the affected directive ambiguous.", "结构不完整会让受影响的指令含义不明确。",
		"Inspect the token and preceding fields, then compare the line with a complete directive of the same kind.", "检查该标记及前面的字段，并与同类指令的完整示例比较。",
		"Add, remove, or delimit only the field required by the intended directive; keep quoting and indentation correct.", "只补充、删除或分隔目标指令所需字段，并保持引号和缩进正确。",
		"2026-01-01 open Assets:Cash", "2026-01-01 open Assets:Cash USD",
		"The missing currency is illustrative; the intended currency belongs to the ledger owner.", "缺少货币只是示例；实际货币由账本所有者决定。",
		"Run orangecount check again.", "再次运行 orangecount check。"),
	"E-PARSE-TOKEN": pair("E-PARSE-TOKEN", PhaseSyntax,
		"Replace the invalid token", "替换无效标记",
		"A token contains characters or punctuation that the grammar cannot accept here.", "标记中的字符或标点不符合当前位置的语法。",
		"The parser cannot safely determine the value or boundary of the directive.", "解析器无法安全确定该指令的值或边界。",
		"Inspect the highlighted token and nearby punctuation, account, amount, or currency spelling.", "检查高亮标记及附近标点、账户、金额或货币拼写。",
		"Replace only the malformed token with a valid value of the intended kind; preserve accounting meaning.", "只将格式错误的标记替换为目标类型的有效值，并保持会计含义。",
		"2026-01-01 open Assets:Cash US$", "2026-01-01 open Assets:Cash USD",
		"The example uses a conventional currency token; use the commodity declared by the ledger.", "示例使用常见货币标记；请使用账本声明的商品。",
		"Run orangecount check again.", "再次运行 orangecount check。"),
	"E-PARSE-STRING": pair("E-PARSE-STRING", PhaseSyntax,
		"Close the string", "闭合字符串",
		"A quoted string reaches the end of its line or file without a closing quote.", "带引号的字符串到行尾或文件尾仍没有结束引号。",
		"The parser cannot know where the narration, metadata value, or other text ends.", "解析器无法确定摘要、元数据值或其他文本在哪里结束。",
		"Find the opening quote and check escaped quotes or an unintended line break.", "找到开始引号，检查转义引号或误产生的换行。",
		"Add the matching quote or correct the intended escaped text; do not move unrelated transaction lines.", "补上匹配的结束引号或修正转义文本；不要移动无关交易行。",
		"2026-01-01 * \"Dinner", "2026-01-01 * \"Dinner\"",
		"The example closes a narration string; confirm the actual narration independently.", "示例只闭合摘要字符串；请独立确认实际摘要。",
		"Run orangecount check again.", "再次运行 orangecount check。"),
	"E-EVAL-OPEN": pair("E-EVAL-OPEN", PhaseSemantic,
		"Review the account opening directives", "检查账户开户指令",
		"An account is opened more than once or with an invalid opening date.", "账户被重复开户，或开户日期无效。",
		"An account must have one valid lifecycle start before postings can use it.", "账户必须先有一个有效的生命周期起点，记账才能使用它。",
		"Inspect all open directives for the account and their dates, including possible spelling variants.", "检查该账户的所有 open 指令及日期，并确认是否存在拼写变体。",
		"Keep one intended open directive and correct its date or spelling; review dependent postings before deleting history.", "保留一个预期开户指令并修正日期或拼写；删除历史前先检查依赖记账。",
		"2026-01-01 open Assets:Cash USD\n2026-01-02 open Assets:Cash USD", "2026-01-01 open Assets:Cash USD",
		"The example removes a duplicate declaration; choose the authoritative opening date from records.", "示例删除了重复声明；请根据记录选择权威开户日期。",
		"Run orangecount check again, then review lifecycle diagnostics.", "再次运行 orangecount check，然后复查生命周期诊断。"),
	"E-EVAL-REOPEN": pair("E-EVAL-REOPEN", PhaseSemantic,
		"Keep the account lifecycle continuous", "保持账户生命周期连续",
		"An account is opened again after it was closed.", "账户在销户后再次开户。",
		"Closing ends that lifecycle; reopening the same name changes the meaning of later postings.", "销户结束该生命周期；同名重开会改变后续记账的含义。",
		"Compare the close date and later open date; decide whether later activity needs a distinct account name.", "比较销户日期和后续开户日期，判断后续活动是否需要不同账户名。",
		"Use one continuous lifecycle or a deliberately distinct account name, preserving the owner's intent.", "保持一个连续生命周期，或选择明确不同的账户名，同时保留所有者的会计意图。",
		"2026-01-01 open Assets:Cash USD\n2026-06-01 close Assets:Cash\n2026-07-01 open Assets:Cash USD", "2026-01-01 open Assets:Cash USD\n2026-06-01 close Assets:Cash",
		"The example removes the later open; it does not decide where later activity belongs.", "示例删除了后续开户，但不替你决定后续活动应归属哪里。",
		"Run orangecount check again.", "再次运行 orangecount check。"),
	"E-EVAL-CLOSE": pair("E-EVAL-CLOSE", PhaseSemantic,
		"Correct the close directive", "修正销户指令",
		"An account close is invalid or occurs after another close.", "销户指令无效，或发生在另一次销户之后。",
		"A close must refer to an open account and occur only once in its lifecycle.", "销户必须指向已开户账户，并且一个生命周期中只能发生一次。",
		"Inspect the account's open and close directives in date order and check exact spelling.", "按日期顺序检查账户的开户和销户指令，并确认账户名拼写一致。",
		"Correct the date or account, or remove only an unintended duplicate after reviewing boundary postings.", "修正日期或账户；误写的重复销户需检查边界记账后再删除。",
		"2026-01-01 open Assets:Cash USD\n2026-06-01 close Assets:Cash\n2026-07-01 close Assets:Cash", "2026-01-01 open Assets:Cash USD\n2026-06-01 close Assets:Cash",
		"The example removes a duplicate close; the intended lifecycle remains an owner decision.", "示例删除重复销户；实际生命周期由账本所有者决定。",
		"Run orangecount check again.", "再次运行 orangecount check。"),
	"E-EVAL-POSTING": pair("E-EVAL-POSTING", PhaseSemantic,
		"Align the posting with the account lifecycle", "让记账符合账户生命周期",
		"A posting uses an account outside its open-to-close lifecycle.", "记账使用了不在账户开户至销户生命周期内的账户。",
		"Posting outside that interval makes account state and reports ambiguous.", "生命周期之外的记账会让账户状态和报表含义不明确。",
		"Compare the posting date with open and close directives and check the exact account name and include file.", "将记账日期与开户、销户指令比较，并检查账户名和 include 文件。",
		"Correct the posting date or account, or add an intended lifecycle directive only when the source record supports it.", "确认源记录后修正记账日期或账户名，或补上确实需要的生命周期指令。",
		"2026-06-01 close Assets:Cash\n2026-07-01 * \"Later\"\n  Assets:Cash 1 USD", "2026-06-01 close Assets:Cash\n2026-07-01 * \"Later\"\n  Assets:Cash 1 USD",
		"The example keeps the line to show the location; choose the correction from the source record.", "示例保留该行以展示位置；应根据源记录选择正确修复。",
		"Run orangecount check and verify the account timeline.", "再次运行 orangecount check，并核对账户时间线。"),
	"E-EVAL-CURRENCY": pair("E-EVAL-CURRENCY", PhaseSemantic,
		"Use an allowed account currency", "使用账户允许的货币",
		"A posting currency is not allowed by the account's open directive.", "记账货币不在账户开户指令允许的货币集合中。",
		"The account's declared currency set constrains which postings belong to it.", "账户声明的货币集合限制哪些记账可以归入该账户。",
		"Inspect the allowed currencies and posting currency; check for a spelling or declaration mismatch.", "查看账户允许的货币和记账货币，确认是拼写错误还是声明不匹配。",
		"Correct the posting or declared currency set only when it matches the accounting record.", "只有符合会计记录时，才修正记账货币或账户声明的货币集合。",
		"2000-01-01 open Assets:Cash USD\n2000-01-02 * \"Exchange\"\n  Assets:Cash 1 EUR", "2000-01-01 open Assets:Cash EUR\n2000-01-02 * \"Exchange\"\n  Assets:Cash 1 EUR",
		"The example changes a declaration for illustration; do not add a currency merely to silence the error.", "示例为说明而修改了声明；不要仅为消除诊断而添加货币。",
		"Run orangecount check and review balancing effects.", "再次运行 orangecount check，并检查平衡影响。"),
	"E-EVAL-UNBALANCED": pair("E-EVAL-UNBALANCED", PhaseSemantic,
		"Review every posting in the transaction", "检查交易中的所有记账",
		"The transaction does not balance for one or more currencies.", "交易在一个或多个货币上没有平衡。",
		"A transaction must account for the complete change in value before a snapshot can be trusted.", "在可信的账本快照发布前，交易必须完整记录价值变化。",
		"Inspect all postings, currencies, costs, and inferred amounts; use the source record to find the missing or misstated leg.", "检查所有记账、货币、成本和省略金额；根据源记录判断缺失或错误的腿。",
		"Add or correct only the posting supported by the source record; do not invent an account or amount.", "只添加或修正源记录支持的记账；不要编造账户或金额。",
		"2000-01-01 * \"Example\"\n  Assets:Cash 10 USD", "2000-01-01 * \"Example\"\n  Assets:Cash 10 USD\n  Expenses:Example -10 USD",
		"The counter-posting is generic; choose the real account and amount from transaction evidence.", "抵消记账只是通用示例；请依据交易证据选择真实账户和金额。",
		"Run orangecount check and confirm no new balance or inventory diagnostics appear.", "再次运行 orangecount check，并确认没有新增余额或库存诊断。"),
	"E-EVAL-INFER": pair("E-EVAL-INFER", PhaseSemantic,
		"Make the omitted amount unambiguous", "让省略金额明确",
		"An omitted posting amount cannot be inferred from the other postings.", "无法从其他记账推断省略的记账金额。",
		"Inference is safe only when one amount remains unambiguous in a single currency context.", "只有在单一货币上下文中剩余一个明确金额时，自动推断才安全。",
		"Inspect every posting and currency and check whether multiple amounts or missing legs prevent inference.", "检查每条记账和货币，确认是否有多个金额或缺失腿阻止推断。",
		"Write the amount explicitly when the source record establishes it, or revise the structure to leave one valid inference.", "源记录确定金额时显式写出；否则调整结构，使其只留下一个有效推断。",
		"2000-01-01 * \"Example\"\n  Assets:Cash 10 USD\n  Expenses:Example", "2000-01-01 * \"Example\"\n  Assets:Cash 10 USD\n  Expenses:Example -10 USD",
		"The explicit amount is illustrative; do not use a guessed remainder as bookkeeping evidence.", "显式金额只是示例；不要把猜出的差额当作记账证据。",
		"Run orangecount check again.", "再次运行 orangecount check。"),
	"E-EVAL-BALANCE": pair("E-EVAL-BALANCE", PhaseSemantic,
		"Reconcile the balance assertion", "核对余额断言",
		"The balance at the assertion date differs from the asserted balance beyond tolerance.", "断言日期的余额超出了允许容差，与断言值不一致。",
		"A balance assertion is an explicit check against ledger history and must not be silently ignored.", "余额断言是对账本历史的显式检查，不能被静默忽略。",
		"Inspect the assertion date, account, currency, expected amount, preceding postings, and configured tolerance.", "检查断言日期、账户、货币、期望金额、之前的记账和配置容差。",
		"Correct the source posting or assertion only after reconciliation; do not change it to hide an unexplained difference.", "与外部记录核对后修正源记账或断言；不要掩盖无法解释的差异。",
		"2000-01-02 balance Assets:Cash 10 USD", "2000-01-02 balance Assets:Cash 10 USD",
		"The snippets show directive shape only; the authoritative amount comes from reconciliation records.", "片段只展示指令形状；权威金额来自对账记录。",
		"Run orangecount check and review any remaining assertion.", "再次运行 orangecount check，并复查剩余断言。"),
	"E-EVAL-INVENTORY": pair("E-EVAL-INVENTORY", PhaseSemantic,
		"Reconcile the available inventory", "核对可用库存",
		"A posting consumes more units or lots than are available at that point.", "记账在当时消耗了超过可用数量或批次的库存。",
		"Inventory booking cannot consume a position that has not been acquired or is unavailable under booking rules.", "库存记账不能消耗尚未取得或违反记账规则的可用头寸。",
		"Inspect earlier acquisitions, reductions, lots, costs, booking mode, and commodity spelling.", "检查之前的取得、减少、批次、成本、记账方式和商品拼写。",
		"Correct the transaction, lot/cost annotation, or booking configuration only when supported; do not invent an acquisition.", "只有源记录支持时才修正交易、批次/成本标注或记账配置；不要编造取得记录。",
		"2000-01-01 * \"Sell\"\n  Assets:Shares -1 HOOL", "2000-01-01 * \"Buy\"\n  Assets:Shares 1 HOOL",
		"The example illustrates acquisition direction; it is not a recommendation to add a trade.", "示例只说明取得方向，不建议添加一笔交易。",
		"Run orangecount check and review the resulting lots.", "再次运行 orangecount check，并复查生成的批次。"),
	"E-EVAL-PAD": pair("E-EVAL-PAD", PhaseSemantic,
		"Make the pad source available", "使 pad 来源可用",
		"A pad directive names a source account that cannot supply the padding entry.", "pad 指令指定的来源账户无法提供补充值。",
		"Padding must come from a valid account in the same evaluated ledger graph.", "补值必须来自同一求值账本图中的有效账户。",
		"Inspect the pad source lifecycle and the later balance assertion intended to consume it.", "检查 pad 来源账户生命周期，以及预期消费它的后续余额断言。",
		"Correct the source account or lifecycle only when supported; remove an accidental unused pad.", "只有源记录支持时才修正来源账户或生命周期；删除误写且未使用的 pad。",
		"2000-01-01 pad Assets:Cash Equity:Opening", "2000-01-01 pad Assets:Cash Equity:Opening\n2000-01-01 open Equity:Opening USD",
		"The added open directive is a shape example; it must match the real source account and currency.", "新增开户指令只是形状示例；必须匹配真实来源账户和货币。",
		"Run orangecount check and confirm a later assertion consumes the pad.", "再次运行 orangecount check，并确认后续断言消费了 pad。"),
	"E-EVAL-TOLERANCE": pair("E-EVAL-TOLERANCE", PhaseSemantic,
		"Correct the balance tolerance", "修正余额容差",
		"A balance assertion tolerance is invalid in its directive or option context.", "余额断言或其选项中的容差无效。",
		"Tolerance controls an explicit comparison and must be a valid non-negative amount in context.", "容差控制显式比较，因此必须是在当前上下文中有效的非负金额。",
		"Inspect the tolerance metadata or option, numeric format, assertion currency, and sign.", "检查容差元数据或选项、数字格式、断言货币和正负号。",
		"Use a valid non-negative tolerance or remove the override to use the documented default.", "使用有效非负容差，或删除覆盖值以使用文档化默认值。",
		"tolerance: -0.01 USD", "tolerance: 0.01 USD",
		"The magnitude is illustrative; choose a tolerance justified by reconciliation policy.", "数值大小只是示例；请根据对账政策选择容差。",
		"Run orangecount check and review the assertion result.", "再次运行 orangecount check，并复查断言结果。"),
	"E-EVAL-OPTION": pair("E-EVAL-OPTION", PhaseSemantic,
		"Correct the option value", "修正选项值",
		"An option is unsupported or its value does not have the expected form.", "选项不受支持，或其值的格式无效。",
		"Options affect parsing or presentation rules; accepting an invalid value makes results ambiguous.", "选项会影响解析或展示规则；接受无效值会让结果含义不明确。",
		"Inspect the option key, value, spelling, quoting, and this release's v3 compatibility boundary.", "检查选项键、值、拼写、引号和当前版本的 v3 兼容边界。",
		"Use a supported value or remove the option after confirming intended behavior.", "确认预期行为后使用支持的值或删除选项。",
		"option \"operating_currency\" \"US$\"", "option \"operating_currency\" \"USD\"",
		"The example fixes a token spelling; use a currency supported by the ledger.", "示例只修正标记拼写；请使用账本支持的货币。",
		"Run orangecount check and inspect reports affected by the option.", "再次运行 orangecount check，并检查受选项影响的报表。"),
}

// Lookup returns a localized immutable copy of the guide for code.
func Lookup(code, locale string) (Guide, bool) {
	value, ok := catalogue[strings.TrimSpace(code)]
	if !ok {
		return Guide{}, false
	}
	if locale == LocaleChinese {
		return clone(value.Chinese), true
	}
	return clone(value.English), true
}

func clone(value Guide) Guide {
	value.Inspect = append([]string(nil), value.Inspect...)
	value.SafeSteps = append([]string(nil), value.SafeSteps...)
	return value
}

// Codes returns all codes with authored guidance in stable order.
func Codes() []string {
	values := make([]string, 0, len(catalogue))
	for code := range catalogue {
		values = append(values, code)
	}
	sort.Strings(values)
	return values
}

// Order returns the repair phase for code. Unknown codes are kept after
// source and syntax problems without claiming an exact causal relationship.
func Order(code string) string {
	if value, ok := catalogue[strings.TrimSpace(code)]; ok {
		return value.English.Phase
	}
	return PhaseSemantic
}

// ValidateCoverage verifies every released error code has complete bilingual
// guidance and that no stale topic remains in the local catalogue.
func ValidateCoverage(releasedErrorCodes []string) error {
	released := make(map[string]bool, len(releasedErrorCodes))
	for _, code := range releasedErrorCodes {
		if !strings.HasPrefix(code, "E-") || code == "" {
			return fmt.Errorf("invalid released diagnostic code %q", code)
		}
		released[code] = true
	}
	for _, code := range releasedErrorCodes {
		value, ok := catalogue[code]
		if !ok {
			return fmt.Errorf("missing repair guidance for %s", code)
		}
		if err := validateGuide(code, LocaleEnglish, value.English); err != nil {
			return err
		}
		if err := validateGuide(code, LocaleChinese, value.Chinese); err != nil {
			return err
		}
	}
	for code := range catalogue {
		if !released[code] {
			return fmt.Errorf("repair guidance has no released diagnostic code: %s", code)
		}
	}
	return nil
}

func validateGuide(code, locale string, value Guide) error {
	fields := map[string]string{
		"topic": value.Topic, "phase": value.Phase, "short_action": value.ShortAction,
		"what": value.What, "why": value.Why, "revalidate": value.Revalidate,
		"example.before": value.Example.Before, "example.after": value.Example.After,
		"example.note": value.Example.Note,
	}
	if value.Code != code {
		return fmt.Errorf("%s %s code mismatch: %q", code, locale, value.Code)
	}
	for field, content := range fields {
		if strings.TrimSpace(content) == "" {
			return fmt.Errorf("%s %s missing %s", code, locale, field)
		}
	}
	if value.Topic != "diagnostics/"+code {
		return fmt.Errorf("%s %s has unstable topic %q", code, locale, value.Topic)
	}
	if len(value.Inspect) == 0 || len(value.SafeSteps) == 0 {
		return fmt.Errorf("%s %s missing inspect or safe steps", code, locale)
	}
	return nil
}
