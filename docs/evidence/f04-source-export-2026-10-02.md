# F04 安全会计源导出实施证据

新增internal/sourceexport的Prepare/Plan.Write边界，CLI export/dialectize改为薄编排。当前设计具体化为入口路径+Options，由模块拥有源捕获、解析/求值与冻结输出；原dialect仍只负责语法转换。未新增原生源导出按钮。

公开自动化验证全局/账户AVERAGE及未登记option/plugin/保留posting metadata拒绝；源、附件、缺失声明路径、symlink、hardlink、旧附件inode与入口目录内输出保护；外部/绝对/重复/glob include拒绝；源更改后旧Plan失效。逆转换可保留AVERAGE源，但仍要求重新加载语义等价；旧export→dialectize→export fixpoint通过。

事务故障覆盖第二文件写失败、部分写后失败、fsync失败、rename失败、父目录移走并替换、并发新目录出现：原目标字节/并发目录身份保持，无半套目标，所属临时树清理。macOS真实排他rename接受成功的新树，已有树拒绝。FinderInfo alias标志的真实文件系统负例在独立Python工具中拒绝且原目标不变。

独立固定Beancount3.2.3工具验证35个公开源导出往返，逐交易共71个完整库存截点；ordered entries包含金额、成本日期/标签、metadata、flags，收支 posting及中间/最终库存一致。包括dialect手写独立期望、跨文件metadata stack、pad、FIFO/LIFO、STRICT/SIZE、HIFO/NONE、总成本、总价格及容差。另6明确拒绝：3 AVERAGE、2入口外include、1 Finder alias目标。无需Python进入产品运行时；不导出私人账本。

最终Go全量通过；sourceexport/documents/CLI race、fmt/vet/license通过；51会计、215查询通过；CLI构建、macOS无cgo及Linux amd64无cgo交叉编译通过，后两者不代表多文件发布已实现。原生源码未因本包改动；F03四套原生检查继承，最终clean App与包另外记录。跨平台无cgo编译不等于多文件事务可用，能力限制见[source-export](../source-export.md)。

最终只读FinanceBook重新执行当前typed-v2真实C/Swift探针与固定Python同源捕获：47文件、双方2836 entries，0错误/缺失/额外/库存差异；Expense73、Income22均相等，native_checked/native_contract/same_input_capture均true；3持仓截点、48价格检查。无私人源导出，未修改原账本。

最终源码5f4fc4c，clean 0.1.5 App构建/签名/bundle检查通过；ZIP仓库外解压与build-info比对通过。App SHA-256 `3369cfd6faa3463ca01dc05bd24df9400e43fc696ebe8211d3b5d1f52bd470fc`。CLI包独立解压、字节比对并实际公开export/v3 check通过，SHA-256 `d525defb0ca2ebfa99163f6ccc958a3f2a6d85329c73283085ca892a88b4255b`。使用说明见[0.1.5交付](../native-development-release-0.1.5.md)。
