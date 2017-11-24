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
	"time"
)

const (
	STATUS_NOT_EXISTS = -1
	STATUS_PENDING    = 0
	STATUS_RUNNING    = 1
	STATUS_FINISHED   = 2
	STATUS_ERROR      = 3
	STATUS_CANCELED   = 4
	STATUS_TIMEOUT    = 5

	WORKFLOW_NIL       = -1
	WORKFLOW_PHYML_SMS = 8
	WORKFLOW_FASTTREE  = 9

	ALIGN_AMINOACIDS = 0
	ALIGN_NUCLEOTIDS = 1
)

type Analysis struct {
	Id    string `json:"id"` // sha256 sum of reftree and boottree files
	EMail string `json:"-"`  // EMail of the job creator, may be empty string ""
	// Four next attributes are for users who want to build the trees using PhyML-SMS of galaxy
	SeqAlign      string `json:"-"`        // Input Fasta Sequence Alignment if user wants to build the ref/boot trees (priority over reffile and bootfile)
	NbootRep      int    `json:"nbootrep"` // Number of bootstrap replicates given by the user to build the bootstrap trees
	Alignfile     string `json:"align"`    // Alignment result file returned by galaxy workflow if users gave a input sequence file
	AlignAlphabet int    `json:"alphabet"` // Alignment alphabet: 0: aa | 1 : nt
	Workflow      int    `json:"workflow"` // The galaxy workflow that has been run. 8:PHYML-SMS, 9: FASTTREE

	Reffile      string `json:"-"`            // reftree original file (to be able to close it)
	Bootfile     string `json:"-"`            // bootstrap original file (to be able to close it)
	FbpTree      string `json:"fbptree"`      // Tree with Fbp supports
	TbeNormTree  string `json:"tbenormtree"`  // resulting newick tree with support
	TbeRawTree   string `json:"tberawtree"`   // result tree with raw <id|avg_dist|depth> as branch names
	TbeLogs      string `json:"tbelogs"`      // log file
	Status       int    `json:"status"`       // status code of the analysis
	Message      string `json:"message"`      // error message if any
	Nboot        int    `json:"nboot"`        // number of trees that have been processed
	StartPending string `json:"startpending"` // Analysis queue time
	StartRunning string `json:"startrunning"` // Analysis Start running time
	End          string `json:"end"`          // Analysis End time
}

func NewAnalysis() (a *Analysis) {
	a = &Analysis{
		Id:           "none",
		EMail:        "",
		SeqAlign:     "",
		NbootRep:     0,
		Alignfile:    "",
		Workflow:     WORKFLOW_NIL,
		Reffile:      "",
		Bootfile:     "",
		FbpTree:      "",
		TbeNormTree:  "",
		TbeRawTree:   "",
		TbeLogs:      "",
		Status:       STATUS_NOT_EXISTS,
		Message:      "",
		Nboot:        0,
		StartPending: "",
		StartRunning: "",
		End:          "",
	}
	return
}

func (a *Analysis) StatusStr() (st string) {
	switch a.Status {
	case STATUS_NOT_EXISTS:
		st = "Analysis does not exist"
	case STATUS_PENDING:
		st = "Pending"
	case STATUS_RUNNING:
		st = "Running"
	case STATUS_FINISHED:
		st = "Finished"
	case STATUS_ERROR:
		st = "Error"
	case STATUS_CANCELED:
		st = "Canceled"
	case STATUS_TIMEOUT:
		st = "Timeout"
	default:
		st = "Unknown"
	}
	return
}

func (a *Analysis) WorkflowStr() string {
	switch a.Workflow {
	case WORKFLOW_PHYML_SMS:
		return "PhyML-SMS"
	case WORKFLOW_FASTTREE:
		return "FastTree"
	case WORKFLOW_NIL:
		return "Bootstrap alone"
	default:
		return "Unknown"
	}
}

func WorkflowConst(workflow string) (w int, err error) {
	switch workflow {
	case "PhyML-SMS":
		w = WORKFLOW_PHYML_SMS
	case "FastTree":
		w = WORKFLOW_FASTTREE
	default:
		err = errors.New(fmt.Sprintf("Phylogenetic workflow does not exist: %s", workflow))
	}
	return
}

func (a *Analysis) DelTemp() {
	var dir string
	if a.SeqAlign != "" {
		if err := os.Remove(a.SeqAlign); err != nil {
			log.Print(err)
		}
		dir = filepath.Dir(a.SeqAlign)
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

// Returns the run time of the analysis from the start pending time
// If end date is not filled yet, takes now(). If some dates have
// format issues: returns "?"
func (a *Analysis) RunTime() string {
	var start, end time.Time
	var delta time.Duration
	var err error

	if start, err = time.Parse(time.RFC1123, a.StartPending); err != nil {
		return "?"
	}

	if a.End == "" {
		end = time.Now()
	} else {
		if end, err = time.Parse(time.RFC1123, a.End); err != nil {
			return "?"
		}
	}

	delta = end.Sub(start).Round(time.Second)
	return delta.String()
}

// Returns the run time of the analysis from the start pending time
// If end date is not filled yet, takes now(). If some dates have
// format issues: returns "?"
func (a *Analysis) RunTimeDuration() (delta time.Duration, err error) {
	var start, end time.Time

	if start, err = time.Parse(time.RFC1123, a.StartPending); err != nil {
		return
	}

	if a.End == "" {
		end = time.Now()
	} else {
		if end, err = time.Parse(time.RFC1123, a.End); err != nil {
			return
		}
	}

	delta = end.Sub(start).Round(time.Second)
	return
}

// A job is considered timed out if its run time is > than the given timeout
// and the given timeout is > 0
//
// A timeout <= 0 means : no time timit on runtime
func (a *Analysis) TimedOut(timeout time.Duration) (timedout bool, err error) {
	timedout = true

	var start, end time.Time
	if start, err = time.Parse(time.RFC1123, a.StartPending); err != nil {
		return
	}
	end = time.Now()

	timedout = timeout != 0 && end.After(start.Add(timeout))
	return
}
