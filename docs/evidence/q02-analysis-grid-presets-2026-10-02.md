# Q02 分析样本网格与查询预设（2026-10-02）

## 查询容器补充

独立公开样本现覆盖多币种库存（USD/EUR/STK/ABC）、相消空库存、nil/空/非空成本标签、零成本、查询 28 位除法、总成本产生的精确 1/3，以及 ExecuteTyped 真正 Rows=2 截断。截断样本沿用真实会话捕获身份，在相同公开账本与文本上降低执行预算；不是修改 returned_rows 假造截断，也不是 native 默认 10000 行预算验收。

新增真实故障输入：单成员展开超过 32MiB（重复字符压缩）、声明容器大小超过 16MiB、存储成员数据位翻转导致 CRC 不符、符号链接成员、非法 UTF-8。

两个公开 red 暴露并修复：

1. Go JSON 的大小写宽松字段匹配允许 `view` 与 `VIEW` 同时出现。读取器增加精确字段名白名单，拒绝大小写别名。
2. 修改 Decimal.Display 或 Approximate，重新生成一致 CSV，仍能通过旧读取器。现在从有界 exact 重建格式并比较完整 Decimal，拒绝误导显示值。库存/金额/成本同样使用此检查。

查询 `1.0 / 3` 实际返回 28 位有限小数，初始 1/3 数学期望不适用于该契约；测试调整为既定查询上下文，另外以公开总成本账本实际验证 rational 1/3。没有改变会计或查询数值语义。

## 查询预设

单一生产定义位于 `internal/query/presets.json`，嵌入 Go；目录返回四个预设，原生模型、Store 和 QueryView 消费该目录。中文/英文名称包含：按月收入、按月支出、账户库存余额、分录来源。

预设覆盖完整快照，没有隐含页面日期筛选、没有自动执行。月收支保留原始记账符号与多币种库存，余额保留完整成本库存；没有伪造单一合计或隐含汇率转换。选择只改草稿；已捕获执行的文本、身份和结果保持原值。

独立公开 presets.bean 含两个月及 USD/EUR 收支、已有完整成本库存。固定 Beancount 3.2.3 / BeanQuery 0.2.0 新增四个 oracle 场景，12 个 corpus 现均注册，151/151 严格差分通过。生产预设与 oracle 查询输入一致性测试防止文本偏离。CLI typed-json 实际执行四个预设；真实 Swift/C Store 验证目录、选择不执行、显式运行及保留前次执行。

## 当前检查与限制

实际通过：分析读取全部针对测试、全量 Go、query/desktop/bridge race、fmt/vet/license、12 corpus oracle-check、151 查询差分、CLI 预设测试、四套 mac-test。app 编译与 GUI 结果另行记录，不从代码推断实际验收。

本批不读取私人账本；旧 FinanceBook 与 GUI 证据仅继承。报表 partial/unknown/page 容器、保存并发与资源性能、期间预设、balance/sum(inventory) 长尾、全指令/附件动作和安装/签名/发行门槛仍待完成。初版四预设不代表完整 Q01/Q02 或 24 项任务完成。

## 本批 app 与 GUI 后续结果

`make mac-test; make app` 串行最终退出 0；arm64、部署目标 13.0、开发 ad-hoc 包检查通过，不是旧系统或发布签名验收。

隔离 `org.orangecount.desktop.qa` GUI：原 presets.bean 在默认 legacy 下因 compound-values.bean 的既有零数量总报价行为产生 E-EVAL-UNBALANCED，未修改该语义；随后以同一公开收支输入去除 compound include 并显式声明 Assets:Cash，生成 ignored build/qa-export/presets-legacy.bean。

实际 GUI 观察：四预设可访问；点击收入只修改文本、仍尚未执行且保存按钮禁用；显式执行返回两个月 USD/EUR 原符号库存；点击支出只改草稿，原收入结果及上下文保留，显示草稿修改提示。支出显式执行前工具报告用户改变页面，最新 AX 为资产负债表，故停止 GUI 输入，未将支出 GUI 执行标为通过。用户介入后的 QA 实例保留，未退出或改动用户页面；原用户应用也未操作。支出执行已有真实 Swift/C 及 CLI 证据，不能替代缺失 GUI 观察。
