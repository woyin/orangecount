// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

func valuationCompletenessText(completeness string) string {
	switch completeness {
	case "complete":
		return "估值完整"
	case "partial":
		return "部分估值"
	case "unavailable":
		return "估值不可用"
	case "empty":
		return "没有需要估值的余额"
	default:
		return "估值状态未知"
	}
}

func valuationReasonText(reason string) string {
	switch reason {
	case "missing-local-quote":
		return "缺少本地报价"
	case "unspecified-quote-currency":
		return "未指定报价币种"
	case "missing-cost":
		return "缺少成本资料"
	case "missing-cost-quote":
		return "缺少成本币种报价"
	default:
		return "估值资料不完整"
	}
}
