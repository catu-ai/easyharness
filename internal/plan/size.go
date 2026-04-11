package plan

const (
	PlanSizeXXS         = "XXS"
	PlanSizeXS          = "XS"
	PlanSizeS           = "S"
	PlanSizeM           = "M"
	PlanSizeL           = "L"
	PlanSizeXL          = "XL"
	PlanSizeXXL         = "XXL"
	placeholderPlanSize = "REPLACE_WITH_PLAN_SIZE"
)

var supportedPlanSizes = []string{
	PlanSizeXXS,
	PlanSizeXS,
	PlanSizeS,
	PlanSizeM,
	PlanSizeL,
	PlanSizeXL,
	PlanSizeXXL,
}

func isSupportedPlanSize(value string) bool {
	switch value {
	case PlanSizeXXS, PlanSizeXS, PlanSizeS, PlanSizeM, PlanSizeL, PlanSizeXL, PlanSizeXXL:
		return true
	default:
		return false
	}
}
