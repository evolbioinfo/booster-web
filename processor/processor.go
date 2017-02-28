package processor

import (
	"github.com/fredericlemoine/booster-web/model"
)

type Processor interface {
	LaunchAnalysis(a *model.Analysis) error
	CancelAnalyses() error
}
