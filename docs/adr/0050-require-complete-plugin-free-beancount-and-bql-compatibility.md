# Require complete plugin-free Beancount and BQL compatibility

2026-10-02用户明确要求无需插件机制，但源加载、查询生态与完整兼容验收必须交付。因此标准兼容目标限定为无用户插件的Beancount3.2.3及BeanQuery0.2.0内置账本查询契约，不能继续把标准缺口列为永久unsupported。plugin声明必须明确诊断，不能静默忽略；核心booking/插值/pad/验证不属于被排除的用户插件。

本决策修订ADR-0046的标准查询子集范围。固定上游实际具有SubqueryTable和EvalPivot，旧文档“上游不提供子查询/PIVOT”的判断不能再用于排除这些功能。完整语法/表/类型/重载清单由上游源码与独立oracle审计，保留独立报表但不能用报表替代标准BQL。通用SQL和第三方Python插件未被要求。

ADR-0045的“every valid v3 file stays valid”仍是尚未达到的目标；只能在本决策的无插件范围与完整验收门槛通过后宣称兼容＋扩展。具体任务及证明边界见[实施计划](../beancount-complete-compatibility-plan-2026-10-02.md)。
