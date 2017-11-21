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

package processor

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/fredericlemoine/booster-web/database"
	"github.com/fredericlemoine/booster-web/model"
	"github.com/fredericlemoine/booster-web/notification"
	"github.com/fredericlemoine/golaxy"
)

// The Galaxy processor launches jobs on a remote galaxy server
//
// It can launch only booster if analysis input files are
type GalaxyProcessor struct {
	runningJobs map[string]*model.Analysis // All running jobs key:job id, value:Job
	galaxy      *golaxy.Galaxy             // Connection to Galaxy
	queue       chan *model.Analysis       // Queue of analyses
	boosterid   string                     // Galaxy ID of booster tool
	phymlid     string                     // Galaxy ID of phyml Workflow
	fasttreeid  string                     // Galaxy ID of fasttree Workflow
	db          database.BoosterwebDB      // Connection to database to save results
	notifier    notification.Notifier      // For email notifications
	lock        sync.RWMutex               // Lock to modify running jobs
}

// It will add the Analysis to the Queue and store it in the database
func (p *GalaxyProcessor) LaunchAnalysis(a *model.Analysis) (err error) {
	if err = p.db.UpdateAnalysis(a); err != nil {
		return
	}
	select {
	case p.queue <- a: // Put a in the channel unless it is full
	default:
		log.Print("Queue is full, cancelling job " + a.Id)
		//Channel full. Discarding value
		a.Status = model.STATUS_CANCELED
		a.End = time.Now().Format(time.RFC1123)
		a.Message = "Computing queue is full, please try again in a few minutes"
		/* Insert analysis */
		err = p.db.UpdateAnalysis(a)
		a.DelTemp()
	}
	return
}

// Initializes the Galaxy Processor
func (p *GalaxyProcessor) InitProcessor(url, apikey, boosterid, phymlid, fasttreeid string, db database.BoosterwebDB, notifier notification.Notifier, queuesize int) {
	p.notifier = notifier
	p.db = db
	p.runningJobs = make(map[string]*model.Analysis)
	p.galaxy = golaxy.NewGalaxy(url, apikey, true)
	p.boosterid = boosterid
	p.phymlid = phymlid
	p.fasttreeid = fasttreeid

	if queuesize == 0 {
		queuesize = RUNNERS_QUEUESIZE_DEFAULT
	}
	if queuesize <= 0 {
		log.Fatal("The queue size must be set to a value >0")
	}
	if queuesize < 100 {
		log.Print("The queue size is <100, it may be a problem for users")
	}
	log.Print("Init galaxy processor")
	log.Print(fmt.Sprintf("Queue size: %d", queuesize))
	log.Print(fmt.Sprintf("Searching Booster tool : %s", boosterid))
	if tools, err := p.galaxy.SearchToolID(boosterid); err != nil {
		log.Fatal(err)
	} else {
		if len(tools) == 0 {
			log.Fatal("No tools correspond to the id: " + boosterid)
		} else {
			p.boosterid = tools[len(tools)-1]
		}
	}
	log.Print(fmt.Sprintf("Booster galaxy tool id: %s", p.boosterid))

	// Searches the PhyML-SMS workflow with given id (checks that it exists)
	if phymlwf, err2 := p.galaxy.GetWorkflowByID(phymlid, true); err2 != nil {
		log.Fatal(err2)
	} else {
		p.phymlid = phymlwf.Id
	}

	log.Print(fmt.Sprintf("PhyML-SMS oneclick galaxy shared workflow id: %s", p.phymlid))

	// Searches the FastTree workflow with given id (checks that it exists)
	if fasttreewf, err3 := p.galaxy.GetWorkflowByID(fasttreeid, true); err3 != nil {
		log.Fatal(err3)
	} else {
		p.fasttreeid = fasttreewf.Id
	}
	log.Print(fmt.Sprintf("FastTree oneclick galaxy shared workflow id: %s", p.fasttreeid))

	p.queue = make(chan *model.Analysis, queuesize)

	// We initialize computing routines
	for cpu := 0; cpu < queuesize; cpu++ {
		go func(cpu int) {
			for a := range p.queue {
				log.Print(fmt.Sprintf("CPU=%d | New analysis, id=%s", cpu, a.Id))
				p.db.UpdateAnalysis(a)
				p.newRunningJob(a)
				err := p.submitToGalaxy(a)
				if err != nil {
					log.Print("Error while submitting to galaxy: " + err.Error())
					a.Status = model.STATUS_ERROR
					a.End = time.Now().Format(time.RFC1123)
					a.Message = err.Error()
				}
				p.rmRunningJob(a)
				p.db.UpdateAnalysis(a)
				if err = p.notifier.Notify(a.StatusStr(), a.Id, a.EMail); err != nil {
					log.Print(err)
				}
			}
			log.Print(fmt.Sprintf("CPU %d : End", cpu))
		}(cpu)
	}
}

func (p *GalaxyProcessor) submitBooster(a *model.Analysis, historyid, reffileid, bootfileid string) (fbptreeid, tbenormtreeid, tberawtreeid, tbelogid string, err error) {
	// We launch the job
	var jobs []string

	tl := p.galaxy.NewToolLauncher(historyid, p.boosterid)
	tl.AddFileInput("ref", reffileid, "hda")
	tl.AddFileInput("boot", bootfileid, "hda")

	_, jobs, err = p.galaxy.LaunchTool(tl)
	if err != nil {
		log.Print("Error while launching booster: " + err.Error())
		return
	}

	if len(jobs) != 1 {
		log.Print("Galaxy Error: No jobs in the list")
		err = errors.New("Galaxy error: No jobs in the list")
		return
	}

	for {
		var state string
		var files map[string]string
		state, files, err = p.galaxy.CheckJob(jobs[0])
		if err != nil {
			return
		}
		switch state {
		case "ok":
			var ok bool
			var id string

			// fbp tree file
			if id, ok = files["fbp_tree"]; !ok {
				err = errors.New("Output file (normalized support tree) not present in the galaxy server")
				return
			}
			fbptreeid = id

			// Normalized suport tree file
			if id, ok = files["tbe_norm_tree"]; !ok {
				err = errors.New("Output file (normalized support tree) not present in the galaxy server")
				return
			}
			tbenormtreeid = id

			// Raw average distance tree file
			if id, ok = files["tbe_raw_tree"]; !ok {
				err = errors.New("Output file (raw distance tree) not present in the galaxy server")
				return
			}
			tberawtreeid = id

			// Log file
			if id, ok = files["tbe_log"]; !ok {
				err = errors.New("Output file (log file) not present in the galaxy server")
				return
			}
			tbelogid = id
			a.End = time.Now().Format(time.RFC1123)
			a.Message = "Finished"
			err = p.db.UpdateAnalysis(a)
			return
		case "queued":
			a.Status = model.STATUS_PENDING
			a.Message = "queued"
			err = p.db.UpdateAnalysis(a)
		case "waiting":
			a.Status = model.STATUS_PENDING
			a.Message = "waiting"
			err = p.db.UpdateAnalysis(a)
		case "running":
			a.Status = model.STATUS_RUNNING
			if a.StartRunning == "" {
				a.StartRunning = time.Now().Format(time.RFC1123)
			}
			a.Message = "running"
			err = p.db.UpdateAnalysis(a)
		case "new":
			a.Status = model.STATUS_PENDING
			a.Message = "New Job"
			err = p.db.UpdateAnalysis(a)
		default:
			err = errors.New("Unkown Job state: " + state)
			a.Status = model.STATUS_ERROR
			log.Print("Job in unknown state: " + state)
			p.db.UpdateAnalysis(a)
			return
		}
		time.Sleep(10 * time.Second)
	}
}

// rawtreeid may be "" if support is classical/FBP
func (p *GalaxyProcessor) submitPhyML(a *model.Analysis, historyid, alignfileid string) (alignmentid, fbptreeid, tbenormtreeid, tberawtreeid, tbelogid string, err error) {
	var wfinvocation *golaxy.WorkflowInvocation
	var wfstate *golaxy.WorkflowStatus
	var phymlwf golaxy.WorkflowInfo

	// We import the workflow
	phymlwf, err = p.galaxy.ImportSharedWorkflow(p.phymlid)
	defer p.galaxy.DeleteWorkflow(phymlwf.Id)
	if err != nil {
		log.Print("Error while importing PhyML-SMS oneclick workflow: " + err.Error())
		return
	}

	// Initializes a launcher
	l := p.galaxy.NewWorkflowLauncher(historyid, phymlwf.Id)
	l.AddFileInput("0", alignfileid, "hda")
	l.AddParameter(5, "support_condition|support", "boot")
	l.AddParameter(5, "support_condition|boot_number", fmt.Sprintf("%d", a.NbootRep))

	if wfinvocation, err = p.galaxy.LaunchWorkflow(l); err != nil {
		log.Print("Error while launching PHYML-SMS oneclick workflow: " + err.Error())
		return
	}

	// Now waits for the end of the execution
	for {
		if wfstate, err = p.galaxy.CheckWorkflow(wfinvocation); err != nil {
			log.Print("Error while checking PHYML-SMS oneclick workflow status : " + err.Error())
			return
		}
		switch wfstate.Status() {
		case "ok":
			if alignmentid, err = wfstate.StepOutputFileId(1, "outputAlignment"); err != nil {
				log.Print("Error while getting alignment file from PHYML-SMS oneclick workflow: " + err.Error())
				return
			}

			if fbptreeid, err = wfstate.StepOutputFileId(5, "output_tree"); err != nil {
				log.Print("Error while getting support tree output file id of PHYML-SMS oneclick workflow: " + err.Error())
				return
			}

			if tbenormtreeid, err = wfstate.StepOutputFileId(5, "tbe_norm_tree"); err != nil {
				log.Print("Error while getting raw distance tree output file id of PHYML-SMS oneclick workflow: " + err.Error())
				return
			}

			if tberawtreeid, err = wfstate.StepOutputFileId(5, "tbe_raw_tree"); err != nil {
				log.Print("Error while getting support tree output file id of PHYML-SMS oneclick workflow: " + err.Error())
				return
			}

			if tbelogid, err = wfstate.StepOutputFileId(5, "tbe_log"); err != nil {
				log.Print("Error while getting booster log file from PHYML-SMS oneclick workflow: " + err.Error())
				return
			}
			a.End = time.Now().Format(time.RFC1123)
			a.Message = "Finished"
			err = p.db.UpdateAnalysis(a)
			return
		case "queued":
			a.Status = model.STATUS_PENDING
			a.Message = "queued"
			if err = p.db.UpdateAnalysis(a); err != nil {
				log.Print("Problem updating job: " + err.Error())
			}
		case "waiting":
			a.Status = model.STATUS_PENDING
			a.Message = "waiting"
			if err = p.db.UpdateAnalysis(a); err != nil {
				log.Print("Problem updating job: " + err.Error())
			}
		case "running":
			a.Status = model.STATUS_RUNNING
			if a.StartRunning == "" {
				a.StartRunning = time.Now().Format(time.RFC1123)
			}
			a.Message = "running"
			if err = p.db.UpdateAnalysis(a); err != nil {
				log.Print("Problem updating job: " + err.Error())
			}
		case "new":
			a.Status = model.STATUS_PENDING
			a.Message = "New Job"
			err = p.db.UpdateAnalysis(a)
		default: // May be "unknown", "deleted", "error" or other...
			err = errors.New("Job state : " + wfstate.Status())
			a.Status = model.STATUS_ERROR
			log.Print("Job in unknown state: " + wfstate.Status())
			p.db.UpdateAnalysis(a)
			return
		}
		time.Sleep(10 * time.Second)
	}
}

func (p *GalaxyProcessor) submitFastTree(a *model.Analysis, historyid, alignfileid string) (alignmentid, fbptreeid, tbenormtreeid, tberawtreeid, tbelogid string, err error) {
	var wfinvocation *golaxy.WorkflowInvocation
	var wfstate *golaxy.WorkflowStatus
	var fasttreewf golaxy.WorkflowInfo

	// We import the workflow
	fasttreewf, err = p.galaxy.ImportSharedWorkflow(p.fasttreeid)
	defer p.galaxy.DeleteWorkflow(fasttreewf.Id)
	if err != nil {
		log.Print("Error while importing FastTree oneclick workflow: " + err.Error())
		return
	}

	// Initializes FastTree launcher
	l := p.galaxy.NewWorkflowLauncher(historyid, fasttreewf.Id)
	l.AddFileInput("0", alignfileid, "hda")
	l.AddParameter(5, "support_condition|support", "boot")
	l.AddParameter(5, "support_condition|boot_number", fmt.Sprintf("%d", a.NbootRep))

	if wfinvocation, err = p.galaxy.LaunchWorkflow(l); err != nil {
		log.Print("Error while launching PHYML-SMS oneclick workflow: " + err.Error())
		return
	}

	// Now waits for the end of the execution
	for {
		if wfstate, err = p.galaxy.CheckWorkflow(wfinvocation); err != nil {
			log.Print("Error while checking PHYML-SMS oneclick workflow status : " + err.Error())
			return
		}
		switch wfstate.Status() {
		case "ok":
			if alignmentid, err = wfstate.StepOutputFileId(1, "outputAlignment"); err != nil {
				log.Print("Error while getting alignment file from FastTree oneclick workflow: " + err.Error())
				return
			}

			if fbptreeid, err = wfstate.StepOutputFileId(4, "output_tree"); err != nil {
				log.Print("Error while getting fbp support tree output file id of FastTree oneclick workflow: " + err.Error())
				return
			}

			if tbenormtreeid, err = wfstate.StepOutputFileId(4, "tbe_norm_tree"); err != nil {
				log.Print("Error while getting support tree output file id of FastTree oneclick workflow: " + err.Error())
				return
			}

			if tberawtreeid, err = wfstate.StepOutputFileId(4, "tbe_raw_tree"); err != nil {
				log.Print("Error while getting raw tree output file id of FastTree oneclick workflow: " + err.Error())
				return
			}

			if tbelogid, err = wfstate.StepOutputFileId(4, "tbe_log"); err != nil {
				log.Print("Error while getting booster log file from FastTree oneclick workflow: " + err.Error())
				return
			}
			a.End = time.Now().Format(time.RFC1123)
			a.Message = "Finished"
			err = p.db.UpdateAnalysis(a)
			return
		case "queued":
			a.Status = model.STATUS_PENDING
			a.Message = "queued"
			if err = p.db.UpdateAnalysis(a); err != nil {
				log.Print("Problem updating job: " + err.Error())
			}
		case "waiting":
			a.Status = model.STATUS_PENDING
			a.Message = "waiting"
			if err = p.db.UpdateAnalysis(a); err != nil {
				log.Print("Problem updating job: " + err.Error())
			}
		case "running":
			a.Status = model.STATUS_RUNNING
			if a.StartRunning == "" {
				a.StartRunning = time.Now().Format(time.RFC1123)
			}
			a.Message = "running"
			if err = p.db.UpdateAnalysis(a); err != nil {
				log.Print("Problem updating job: " + err.Error())
			}
		case "new":
			a.Status = model.STATUS_PENDING
			a.Message = "New Job"
			if err = p.db.UpdateAnalysis(a); err != nil {
				log.Print("Problem updating job: " + err.Error())
			}
		default: // May be "unknown", "deleted", "error" or other...
			err = errors.New("Job state : " + wfstate.Status())
			a.Status = model.STATUS_ERROR
			log.Print("Job in unknown state: " + wfstate.Status())
			p.db.UpdateAnalysis(a)
			return
		}
		time.Sleep(10 * time.Second)
	}
}

func (p *GalaxyProcessor) submitToGalaxy(a *model.Analysis) (err error) {
	var outcontent []byte
	var reffileid string
	var bootfileid string
	var seqid string
	var alignid, fbptreeid, tbenormtreeid, tberawtreeid, tbelogid string
	var history golaxy.HistoryFullInfo

	// We create an history
	history, err = p.galaxy.CreateHistory("Booster History")
	// And we delete the history
	defer p.galaxy.DeleteHistory(history.Id)
	if err != nil {
		log.Print("Error while Creating History: " + err.Error())
		return
	}

	log.Print("History: " + history.Id)

	// If we have a sequence file, then we build the trees from it
	// and compute supports using the PHYML-SMS oneclick workflow from galaxy
	if a.SeqFile != "" {
		if a.Workflow == model.WORKFLOW_NIL {
			err = errors.New("Phylogenetic workflow to launch is not defined")
			log.Print("Error while Uploading reference sequence file: " + err.Error())
			return
		}
		// We upload ref sequence file to history
		seqid, _, err = p.galaxy.UploadFile(history.Id, a.SeqFile, "fasta")
		if err != nil {
			log.Print("Error while Uploading reference sequence file: " + err.Error())
			return
		}

		if a.Workflow == model.WORKFLOW_PHYML_SMS {
			if alignid, fbptreeid, tbenormtreeid, tberawtreeid, tbelogid, err = p.submitPhyML(a, history.Id, seqid); err != nil {
				log.Print("Error while launching PhyML-SMS oneclick workflow : " + err.Error())
				return
			}
		} else if a.Workflow == model.WORKFLOW_FASTTREE {
			if alignid, fbptreeid, tbenormtreeid, tberawtreeid, tbelogid, err = p.submitFastTree(a, history.Id, seqid); err != nil {
				log.Print("Error while launching FastTree oneclick workflow : " + err.Error())
				return
			}
		} else {
			err = errors.New("Error while launching oneclick workflow, unkown workflow")
			log.Print(err.Error())
			return

		}

		// And we download alignment file
		if outcontent, err = p.galaxy.DownloadFile(history.Id, alignid); err != nil {
			log.Print("Error while downloading alignment file: " + err.Error())
			return
		}
		a.Alignfile = string(outcontent)

	} else if a.Reffile != "" && a.Bootfile != "" {
		// Otherwise we upload the given ref and boot files
		// We upload ref tree to history
		reffileid, _, err = p.galaxy.UploadFile(history.Id, a.Reffile, "nhx")
		if err != nil {
			log.Print("Error while Uploading ref tree file: " + err.Error())
			return
		}

		// We upload boot tree to history
		bootfileid, _, err = p.galaxy.UploadFile(history.Id, a.Bootfile, "nhx")
		if err != nil {
			log.Print("Error while Uploading boot tree file: " + err.Error())
			return
		}
		// Now we submit the booster tool
		fbptreeid, tbenormtreeid, tberawtreeid, tbelogid, err = p.submitBooster(a, history.Id, reffileid, bootfileid)
	} else {
		log.Print("No Reference tree or Bootstrap tree given")
		err = errors.New("No Reference tree or Bootstrap tree given")
		return
	}

	// And we download resulting files
	if outcontent, err = p.galaxy.DownloadFile(history.Id, fbptreeid); err != nil {
		log.Print("Error while downloading fbp tree file: " + err.Error())
		return
	}
	a.FbpTree = string(outcontent)

	if outcontent, err = p.galaxy.DownloadFile(history.Id, tbenormtreeid); err != nil {
		log.Print("Error while downloading support file: " + err.Error())
		return
	}
	a.TbeNormTree = string(outcontent)

	if outcontent, err = p.galaxy.DownloadFile(history.Id, tberawtreeid); err != nil {
		log.Print("Error while downloading avg dist tree file: " + err.Error())
		return
	}
	a.TbeRawTree = string(outcontent)

	if outcontent, err = p.galaxy.DownloadFile(history.Id, tbelogid); err != nil {
		log.Print("Error while downloading log file: " + err.Error())
		return
	}
	a.TbeLogs = string(outcontent)

	a.Status = model.STATUS_FINISHED
	p.db.UpdateAnalysis(a)

	return
}

// Keep track of currently running jobs.
// In order to cancel them when the server stops
func (p *GalaxyProcessor) newRunningJob(a *model.Analysis) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.runningJobs[a.Id] = a
}

func (p *GalaxyProcessor) rmRunningJob(a *model.Analysis) {
	p.lock.Lock()
	defer p.lock.Unlock()

	delete(p.runningJobs, a.Id)
}

func (p *GalaxyProcessor) allRunningJobs() []*model.Analysis {
	p.lock.RLock()
	defer p.lock.RUnlock()
	v := make([]*model.Analysis, 0)
	for _, value := range p.runningJobs {
		v = append(v, value)
	}
	return v
}

func (p *GalaxyProcessor) CancelAnalyses() (err error) {
	for _, a := range p.allRunningJobs() {
		log.Print("Cancelling job : " + a.Id)
		a.Status = model.STATUS_CANCELED
		a.End = time.Now().Format(time.RFC1123)
		a.Message = "Canceled after a server restart"
		if err = p.db.UpdateAnalysis(a); err != nil {
			log.Print(err)
		}
	}
	return
}
