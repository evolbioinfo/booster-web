package database

import (
	"github.com/fredericlemoine/booster-web/model"
)

type BoosterwebDB interface {
	GetAnalysis(id string) (*model.Analysis, error)
	UpdateAnalysis(*model.Analysis) error
	Connect() error
	Disconnect() error
}
