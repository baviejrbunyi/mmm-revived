package server

import "mmm-revived/internal/model"

func init() {
	components["manualComp"] = field(func(a *model.Avatar) model.ManualComponent { return a.Manual })
	components["benefitsSummaryComp"] = field(func(a *model.Avatar) model.BenefitsComponent { return a.Benefits })
}
