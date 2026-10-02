# 缺少功能实施状态

基线：0.1.1源码278c6e4，设计4be0846。本文件记录本轮新增证据；其他历史检查不可自动覆盖当前代码。当前源码尚未冻结提交或生成对应开发包。

| 任务 | 代码状态 | 公开输入 | 当前检查 | GUI | 提交/包 | 外部缺口 |
| --- | --- | --- | --- | --- | --- | --- |
| P00 | 完成当前矩阵入口与Module核对 | 现有reader/DTO/export | 当前旧状态已标历史 | 不适用 | 待冻结 | 无 |
| F01.1 | v2 reader/writer、record/set/list、版本分派已实现 | analysis-v2四样例及恶意ZIP | 独立样例、类型/CRC/成员/预算；查询reader原案例通过 | 不适用 | 待冻结 | 无 |
| F01.2 | 投影身份、16项/16MiB缓存、树Adapter已实现 | 简单12.34 USD；valuation | 精确/身份/淘汰/公开往返通过 | 资产负债表已实际保存并reader读回 | 待冻结 | 无 |
| F01.3 | 协议3、C/Swift/Store、通用保存入口已实现 | cashflow及真实临时账本 | 真实Swift/C捕获/ZIP/写入；Store重载冻结、源保护/关闭拒绝通过 | 首批保存成功；其他操作待当前包 | 待冻结 | 无 |
| F01.4 | 趋势/账户Adapter已实现 | history/account-detail | Go往返、真实C/Swift路由捕获通过 | 待当前包 | 待冻结 | 无 |
| F01.5 | 交易/价格Adapter已实现 | cashflow/prices，第二页 | 双单位计数、选择supplement、脱敏及往返通过 | 待当前包 | 待冻结 | 无 |
| F01.6 | 持仓组/批次/历史/归约Adapter已实现 | holdings-trades/unknowns/average-history | 完整成本键、nested page及往返；真实C/Swift首批通过 | 待当前包 | 待冻结 | 无 |
| F01.7 | 收尾中 | 上述全部 | Go全量/race首轮通过；原生最终构建中 | 尚未完成各入口操作 | 未交付 | 无 |
| F02.1–F02.3 | 待实施 | 计划公开根/权限场景 | 未执行 | 未执行 | 无 | 无 |
| F03.0–F03.7 | 调查开始，未改求值行为 | 锁定BeanQuery源码 | balance源码定位；尚无新增差分证据 | 未执行 | 无 | 无 |
| F04.1–F04.4 | 待实施 | 计划公开export场景 | 直接写入路径已定位；尚未修复或验收 | CLI范围 | 无 | 无 |

首批实际GUI：独立org.orangecount.desktop.reportqa读取公开cashflow.bean，资产负债表导出菜单→系统sheet→build/report-public-gui-20261002.ocanalysis，界面显示保存成功；analysisverify输出format_version2、balance_sheet、2行/12列、未截断。该QA为首批构建，不代表之后新增全部Adapter的最终GUI验收。

边界：完整N04/R01及远端/旧系统/签名仍按总计划保留。这里的“已实现”不同于整个功能包退出条件满足。
