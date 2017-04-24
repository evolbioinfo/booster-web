package model

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const (
	STATUS_NOT_EXISTS = -1
	STATUS_PENDING    = 0
	STATUS_RUNNING    = 1
	STATUS_FINISHED   = 2
	STATUS_ERROR      = 3
	STATUS_CANCELED   = 4
	STATUS_TIMEOUT    = 5

	ALGORITHM_BOOSTER   = 6
	ALGORITHM_CLASSICAL = 7
)

type Analysis struct {
	Id           string `json:"id"`           // sha256 sum of reftree and boottree files
	Reffile      string `json:"-"`            // reftree original file (to be able to close it)
	Bootfile     string `json:"-"`            // bootstrap original file (to be able to close it)
	Result       string `json:"result"`       // resulting newick tree with support
	Status       int    `json:"status"`       // status code of the analysis
	Algorithm    int    `json:"algorithm"`    // Algorithm : ALGORITHM_CLASSICAL or ALGORITHM_BOOSTER
	StatusStr    string `json:"statusstr"`    // status of the analysis (string)
	Message      string `json:"message"`      // error message if any
	Nboot        int    `json:"nboot"`        // number of trees that have been processed
	Collapsed    string `json:"collapsed"`    // Newick representation of collapsed resulting tree if any
	StartPending string `json:"startpending"` // Analysis queue time
	StartRunning string `json:"startrunning"` // Analysis Start running time
	End          string `json:"end"`          // Analysis End time
}

func StatusStr(status int) string {
	switch status {
	case STATUS_NOT_EXISTS:
		return "Analysis does not exist"
	case STATUS_PENDING:
		return "Pending"
	case STATUS_RUNNING:
		return "Running"
	case STATUS_FINISHED:
		return "Finished"
	case STATUS_ERROR:
		return "Error"
	case STATUS_CANCELED:
		return "Canceled"
	case STATUS_TIMEOUT:
		return "Timeout"
	default:
		return "Unknown"
	}
}

func AlgorithmStr(algorithm int) string {
	switch algorithm {
	case ALGORITHM_BOOSTER:
		return "booster"
	case ALGORITHM_CLASSICAL:
		return "classical"
	default:
		return "unknown"
	}
}

func AlgorithmConst(algorithm string) (int, error) {
	switch algorithm {
	case "booster":
		return ALGORITHM_BOOSTER, nil
	case "classical":
		return ALGORITHM_CLASSICAL, nil
	default:
		return -1, errors.New(fmt.Sprintf("Algorithm %s does not exist", algorithm))
	}
}

func (a *Analysis) DelTemp() {
	if err := os.Remove(a.Reffile); err != nil {
		log.Print(err)
	}
	if err := os.Remove(a.Bootfile); err != nil {
		log.Print(err)
	}
	if err := os.Remove(filepath.Dir(a.Bootfile)); err != nil {
		log.Print(err)
	}
}
