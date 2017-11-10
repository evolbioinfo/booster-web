/*

BOOSTER-WEB: Web interface to BOOSTER (https://github.com/evolbioinfo/booster)
Alternative method to compute bootstrap branch supports in large trees.

Copyright (C) 2017 BOOSTER-WEB dev team

This program is free software; you can redistribute it and/or
modify it under the terms of the GNU General Public License
as published by the Free Software Foundation; either version 2
of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program; if not, write to the Free Software
Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.

*/

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
	Id string `json:"id"` // sha256 sum of reftree and boottree files

	// Three next attributes are for users who want to build the trees using PhyML-SMS of galaxy
	SeqFile   string `json:"-"`        // Input Fasta Sequence File if user wants to build the ref/boot trees (priority over reffile and bootfile)
	NbootRep  int    `json:"nbootrep"` // Number of bootstrap replicates given by the user to build the bootstrap trees
	Alignfile string `json:"align"`    // Alignment result file returned by galaxy workflow if users gave a input sequence file

	Reffile      string `json:"-"`            // reftree original file (to be able to close it)
	Bootfile     string `json:"-"`            // bootstrap original file (to be able to close it)
	Result       string `json:"result"`       // resulting newick tree with support
	RawTree      string `json:"rawtree"`      // result tree with raw <id|avg_dist|depth> as branch names
	ResLogs      string `json:"reslogs"`      // log file
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
	var dir string
	if a.SeqFile != "" {
		if err := os.Remove(a.SeqFile); err != nil {
			log.Print(err)
		}
		dir = filepath.Dir(a.SeqFile)
	}
	if a.Reffile != "" {
		if err := os.Remove(a.Reffile); err != nil {
			log.Print(err)
		}
		dir = filepath.Dir(a.Reffile)
	}
	if a.Bootfile != "" {

		if err := os.Remove(a.Bootfile); err != nil {
			log.Print(err)
		}
		dir = filepath.Dir(a.Bootfile)
	}
	if dir != "" {
		if err := os.Remove(dir); err != nil {
			log.Print(err)
		}
	}

}
