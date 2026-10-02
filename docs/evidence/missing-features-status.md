# 缺少功能实施状态

基线：0.1.1源码278c6e4，设计4be0846。本文件记录本轮新增证据；其他历史检查不可自动覆盖当前代码。F01源码已冻结280881c并生成0.1.2开发包；实际证据见F01验收文档。

| 任务 | 代码状态 | 公开输入 | 当前检查 | GUI | 提交/包 | 外部缺口 |
| --- | --- | --- | --- | --- | --- | --- |
| P00 | 完成当前矩阵入口与Module核对 | 现有reader/DTO/export | 当前旧状态已标历史 | 不适用 | 280881c / 0.1.2 | 无 |
| F01.1 | v2 reader/writer、record/set/list、版本分派已实现 | analysis-v2四样例及恶意ZIP | 独立样例、类型/CRC/成员/预算；查询reader原案例通过 | 不适用 | 280881c / 0.1.2 | 无 |
| F01.2 | 投影身份、16项/16MiB缓存、树Adapter已实现 | 简单12.34 USD；valuation | 精确/身份/淘汰/公开往返通过 | 资产负债表已实际保存并reader读回 | 280881c / 0.1.2 | 无 |
| F01.3 | 协议3、C/Swift/Store、通用保存入口已实现 | cashflow及真实临时账本 | 真实Swift/C捕获/ZIP/写入；Store重载冻结、源保护/关闭拒绝通过 | 首批保存成功；其他操作主页面实际保存并独立读回；子section自动/C覆盖 | 280881c / 0.1.2 | 无 |
| F01.4 | 趋势/账户Adapter已实现 | history/account-detail | Go往返、真实C/Swift路由捕获通过 | 主页面实际保存并独立读回；子section自动/C覆盖 | 280881c / 0.1.2 | 无 |
| F01.5 | 交易/价格Adapter已实现 | cashflow/prices，第二页 | 双单位计数、选择supplement、脱敏及往返通过 | 主页面实际保存并独立读回；子section自动/C覆盖 | 280881c / 0.1.2 | 无 |
| F01.6 | 持仓组/批次/历史/归约Adapter已实现 | holdings-trades/unknowns/average-history | 完整成本键、nested page及往返；真实C/Swift首批通过 | 主页面实际保存并独立读回；子section自动/C覆盖 | 280881c / 0.1.2 | 无 |
| F01.7 | 收尾中 | 上述全部 | Go全量/race首轮通过；最终原生检查通过 | 7种主页面保存/reader及取消已验；嵌套子入口GUI待补 | 280881c / 0.1.2 | 无 |
| F02.1–F02.3 | 策略/协议4/bookmark/设置/广播已实现 | 临时公开ledger与外部receipts | 全量/race及真实C/Swift双窗口、恢复/替换通过 | 授权、实际Open与撤销禁用通过；按钮辅助功能修订待最终包 | 待冻结 | 旧系统未验 |
| F03.0–F03.7 | 已实施，明确排除项见矩阵 | 16语料/215查询 | 固定oracle、全量/race/native通过 | balance保存、错误定位、sum通过；完整N04未关闭 | 0.1.4构建通过，打包待记录 | f03-query-longtail证据 |
| F04.1–F04.4 | 待实施 | 计划公开export场景 | 直接写入路径已定位；尚未修复或验收 | CLI范围 | 无 | 无 |

首批实际GUI：独立org.orangecount.desktop.reportqa读取公开cashflow.bean，资产负债表导出菜单→系统sheet→build/report-public-gui-20261002.ocanalysis，界面显示保存成功；analysisverify输出format_version2、balance_sheet、2行/12列、未截断。该QA为首批构建，不代表之后新增全部Adapter的最终GUI验收。

边界：完整N04/R01及远端/旧系统/签名仍按总计划保留。这里的“已实现”不同于整个功能包退出条件满足。
