# F01 报表分析文件实施与验收

本轮新增实现：报表v2容器及严格reader/writer、analysisverify版本分派；树报表、趋势、账户概览/历史、交易posting页、价格目录/历史、持仓组/批次/历史/归约投影Adapter；Session固定捕获及16项/16MiB缓存；原生协议3、C入口、Store准备/原子保存及通用菜单。

## 已执行检查

- Go全量首轮通过，包含新analysis Module和全部旧查询reader案例。analysis/desktop/bridge race通过；51会计与163查询、fmt/vet/license通过。取消控制及身份修复后又执行相关race和最终Go全量。
- 原生四套首轮通过：真实Swift/C验证树报表exact CSV、实际ZIP和原子保存，各页C路由捕获；真实Store验证重载不改变捕获字节、源保护与关闭拒绝。最终取消控制与新身份字段的mac-check已exit0，四套及app/bundle通过；查询v1和旧原生回归保持。
- 独立公开样例：empty、multicurrency、partial、paged-postings，保留1/3 exact、零成本/空label、NULL、交易数1与posting数3。它们由独立手工结构/标准CSV编码给出，不用生产writer生成期望。
- 负例：duplicate JSON/member/column、unknown字段/request、非法type/exact/display/nullable、CSV不一致、坏CRC、symlink ZIP member、压缩预算、unordered set及record schema。Writer取消在实际输出时触发并返回context.Canceled，预取消不写字节；原生控制调用验证cancelled错误。
- 页面捕获身份包含snapshot、实际期间、section、排序和有效offset/limit；测试同时保留两页身份，防止后页替换前页捕获。
- 公开各投影→ZIP→reader往返通过，解包JSON无源path字段。归约/批次嵌套列表及page用list<record>保留，来源以opaque ID替代绝对路径。

## GUI已观测

首批独立reportqa读取公开cashflow.bean，实际资产负债表菜单保存至build/report-public-gui-20261002.ocanalysis，界面显示成功；独立analysisverify读回version2、2行12列、未截断。此QA构建早于最终全部Adapter/取消补丁，最终GUI尚需用新实例执行。

## 当前限制

账本与附件保持只读。查询容器仍是v1，报表v2；旧reader不支持v2。仅单次捕获本投影，当前页不代表全集；嵌套批次/贡献列表也各自保留分页。文件不含源账本或附件副本，离线来源ID不能打开原源文件。

尚未完成F02根配置、F03新增查询语义、F04源导出约束；完整N04/R01及发行任务保留。F01最终包、私人当前archive只读对照和各GUI入口记录在本文件后续追加，未执行项不能记通过。

## 当前冻结与实际保存（源码280881c）

最终独立reportfinalqa在公开cashflow.bean上实际保存资产负债表、损益、日记账、仪表盘、持仓、商品价格、账户概览共7个v2文件，界面均显示成功。独立analysisverify全部读回；日记账2交易/4posting，持仓unknown为unavailable。账户CSV与分析文件data.csv逐字节相同。实际取消sheet显示“已取消保存”，目标文件不存在。嵌套持仓/价格子section本轮自动与真实C覆盖，未逐项声称GUI验收完成。

当前native archive只读FinanceBook对照通过：47文件、双方2836分录、0错误/额外/缺失/库存差异，Expenses73与Income22账户一致；native_checked/native_contract为true，3个持仓截至点与48项价格检查通过。未修改私人账本。

0.1.2开发包对应280881c3f4c524e256684d1e7d029ba127318ecd、working_tree=clean。ZIP在仓库外解压后签名/架构/deployment/资源验证通过，build-info与原包一致。SHA-256：`eb5baca9250d6a777eb1544cedd5942377b331e81bcc7b2376f4136491a764f0`。这是ad hoc arm64开发包，旧系统实际兼容与完整安装验收仍未完成。
