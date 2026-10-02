# F03 查询长尾实施证据

16公开语料/215查询，通过固定Beancount3.2.3、BeanQuery0.2.0的独立oracle检查；原163查询保留。新增balance-stream31、balance-cost8、source-order7、aggregate-longtail6。完整成本身份、NULL/空Inventory、源顺序、GROUP/ORDER与数字边界见能力矩阵。

当前Go全量、query/desktop/bridge race、fmt/vet/license通过；原生四套检查及App构建通过。真实C/Swift覆盖balance重复列1000/700、CSV、sum1700、非法分组与byte range。公开GUI保存analysis，独立reader读取两列两行；旧typed-v1文件在升级typed-v2后仍读取通过。

GUI非法GROUP曾两次触发SwiftUI辅助功能递归并实际崩溃；删除错误Text重复自定义accessibilityLabel后，同一错误正常显示且定位account，UTF-8错误定位unknown_fn以及随后sum1700均通过。完整N04未关闭。

本轮只读FinanceBook当前archive：47文件、2836 entries双方一致，错误/entry/inventory差异均0，Expense73、Income22，native_checked与native_contract均true；3持仓截点、48报价。该证据输入会计及查询数学与最终实现相同；后续仅查询契约名称与UI错误标签改变。未导出或提交私人源。

未对齐无效重复include、posting保留filename/lineno加载诊断；任意Python Decimal陷阱、无限指数与任意Any重载未承诺。源导出兼容门必须独立处理这些边界。
