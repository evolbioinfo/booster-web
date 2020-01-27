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
	"math"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/evolbioinfo/booster-web/database"
	"github.com/evolbioinfo/booster-web/model"
	"github.com/evolbioinfo/booster-web/notification"
	"github.com/evolbioinfo/goalign/align"
	"github.com/evolbioinfo/gotree/io/newick"
	"github.com/evolbioinfo/gotree/tree"
	"github.com/fredericlemoine/golaxy"
)

// The Galaxy processor launches jobs on a remote galaxy server
//
// It can launch only booster if analysis input files are
type GalaxyProcessor struct {
	runningJobs map[string]*model.Analysis // All running jobs key:job id, value:Job

	galaxy     *golaxy.Galaxy        // Connection to Galaxy
	queue      chan *model.Analysis  // Queue of analyses
	boosterid  string                // Galaxy ID of booster tool
	phymlid    string                // Galaxy ID of phyml Workflow
	fasttreeid string                // Galaxy ID of fasttree Workflow
	db         database.BoosterwebDB // Connection to database to save results
	notifier   notification.Notifier // For email notifications
	lock       sync.RWMutex          // Lock to modify running jobs
	timeout    int                   // Timeout in seconds: jobs are timedout after this time
	memlimit   int                   // Memory limit for jobs in Bytes. If jobs are estimated to consume more, they are not launched
	queuesize  int                   // Max queue size
	stopping   bool                  // If the server is stopping
}

// It will add the Analysis to the Queue and store it in the database
func (p *GalaxyProcessor) LaunchAnalysis(a *model.Analysis) (err error) {
	if err = p.db.UpdateAnalysis(a); err != nil {
		return
	}
	full := false
	if len(p.runningJobs) >= p.queuesize {
		full = true
	}
	select {
	case p.queue <- a: // Put a in the channel unless it is full
	default:
		full = true
	}

	if full {
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
func (p *GalaxyProcessor) InitProcessor(url, apikey, boosterid, phymlid, fasttreeid string, galaxyrequestattempts int, db database.BoosterwebDB, notifier notification.Notifier, queuesize, timeout, memlimit int) {

	var tool golaxy.ToolInfo
	var err error

	p.stopping = false
	p.notifier = notifier
	p.db = db
	p.runningJobs = make(map[string]*model.Analysis)
	p.galaxy = golaxy.NewGalaxy(url, apikey, true)
	p.galaxy.SetNbRequestAttempts(galaxyrequestattempts)
	p.boosterid = boosterid
	p.phymlid = phymlid
	p.fasttreeid = fasttreeid
	p.timeout = timeout
	p.memlimit = memlimit

	if queuesize == 0 {
		queuesize = RUNNERS_QUEUESIZE_DEFAULT
	}
	if queuesize <= 0 {
		log.Fatal("The queue size must be set to a value >0")
	}
	if queuesize < 100 {
		log.Print("The queue size is <100, it may be a problem for users")
	}
	p.queuesize = queuesize
	log.Print("Init galaxy processor")
	log.Print(fmt.Sprintf("Job timeout: %d", p.timeout))
	log.Print(fmt.Sprintf("Job mem limit: %d", p.memlimit))
	log.Print(fmt.Sprintf("Queue size: %d", queuesize))
	log.Print(fmt.Sprintf("Searching Booster tool : %s", boosterid))
	if tool, err = p.galaxy.GetToolById(boosterid); err != nil {
		log.Fatal(err)
	}
	p.boosterid = tool.Id

	log.Print(fmt.Sprintf("Booster galaxy tool id: %s", p.boosterid))

	// Searches the PhyML-SMS workflow with given id (checks that it exists)
	if tool, err = p.galaxy.GetToolById(phymlid); err != nil {
		log.Fatal("Error while getting phyml workflow id: " + err.Error())
	}
	p.phymlid = tool.Id

	log.Print(fmt.Sprintf("PhyML-SMS galaxy tool id: %s", p.phymlid))

	// Searches the FastTree workflow with given id (checks that it exists)
	if tool, err = p.galaxy.GetToolById(fasttreeid); err != nil {
		log.Fatal("Error while getting fasttree workflow id: " + err.Error())
	}
	p.fasttreeid = tool.Id

	log.Print(fmt.Sprintf("FastTree galaxy tool id: %s", p.fasttreeid))

	p.queue = make(chan *model.Analysis, queuesize)

	// We initialize launching go routine
	p.initJobLauncher()
	// We initialize job monitoring go routine
	p.initJobMonitor()
	// We restore already running jobs on galaxy
	p.restoreRunningJobs()
}

func (p *GalaxyProcessor) submitBooster(a *model.Analysis, reffileid, bootfileid string) (err error) {
	// We launch the job
	var jobs []string

	tl := p.galaxy.NewToolLauncher(a.GalaxyHistory, p.boosterid)
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

	a.JobId = jobs[0]
	p.db.UpdateAnalysis(a)
	return
}

func (p *GalaxyProcessor) checkJob(a *model.Analysis) (state, fbptreeid, tbenormtreeid, tberawtreeid, tbelogid string, err error) {
	var files map[string]string
	var fbptreename, tbenormtreename, tberawtreename, tbelogname string

	// Now check status of galaxy job
	if a.JobId == "" {
		err = errors.New("Galaxy Job ID not already assigned for " + a.Id)
		log.Print(err.Error())
		return
	}

	// Now check status of galaxy job
	if state, files, err = p.galaxy.CheckJob(a.JobId); err != nil {
		log.Print("Error while checking " + a.WorkflowStr() + " workflow status : " + err.Error())
		return
	}

	if a.SeqAlign == "" {
		fbptreename = "fbp_tree"
	} else {
		fbptreename = "out_tree"
	}
	tbenormtreename = "tbe_norm_tree"
	tberawtreename = "tbe_raw_tree"
	tbelogname = "tbe_log"

	switch state {
	case "ok":
		var ok bool
		a.Status = model.STATUS_FINISHED
		a.Message = "Finished"

		// Get result file ids
		if fbptreeid, ok = files[fbptreename]; !ok {
			err = errors.New("Error while getting support tree output file id of workflow " + a.Id)
			log.Print(err.Error())
			state = "error"
			a.Message = err.Error()
			a.Status = model.STATUS_ERROR
		} else if tbenormtreeid, ok = files[tbenormtreename]; !ok {
			err = errors.New("Error while getting raw distance tree output file id of workflow" + a.Id)
			log.Print(err.Error())
			a.Message = err.Error()
			state = "error"
			a.Status = model.STATUS_ERROR
		} else if tberawtreeid, ok = files[tberawtreename]; !ok {
			err = errors.New("Error while getting raw distance tree output file id of workflow" + a.Id)
			log.Print(err.Error())
			a.Message = err.Error()
			state = "error"
			a.Status = model.STATUS_ERROR
		} else if tbelogid, ok = files[tbelogname]; !ok {
			err = errors.New("Error while getting tbe log file id workflow " + a.Id)
			log.Print(err.Error())
			a.Message = err.Error()
			state = "error"
		}
		a.End = time.Now().Format(time.RFC1123)
	case "queued":
		a.Status = model.STATUS_PENDING
		a.Message = "queued"
	case "waiting":
		a.Status = model.STATUS_PENDING
		a.Message = "waiting"
	case "running":
		a.Status = model.STATUS_RUNNING
		if a.StartRunning == "" {
			a.StartRunning = time.Now().Format(time.RFC1123)
		}
		a.Message = "running"
	case "new":
		a.Status = model.STATUS_PENDING
		a.Message = "New Job"
	default: // May be "unknown", "deleted", "error" or other...
		err = errors.New("Job state : " + state)
		a.Status = model.STATUS_ERROR
		a.Message = "Galaxy Error"
		log.Print("Job in unknown state: " + state)
	}

	return
}

func (p *GalaxyProcessor) submitPhyML(a *model.Analysis, alignfileid string) (err error) {
	var jobs []string

	tl := p.galaxy.NewToolLauncher(a.GalaxyHistory, p.phymlid)
	tl.AddFileInput("input_align", alignfileid, "hda")

	if a.AlignAlphabet == model.ALIGN_AMINOACIDS {
		tl.AddParameter("sequence|seqtype", "aa")
	} else if a.AlignAlphabet == model.ALIGN_NUCLEOTIDS {
		tl.AddParameter("sequence|seqtype", "nt")
	} else {
		err = errors.New("Unkown sequence alphabet in alignment")
		return
	}
	tl.AddParameter("stat_crit", "aic")
	tl.AddParameter("move", "SPR")
	tl.AddParameter("bootstrap|support", "boot")
	tl.AddParameter("bootstrap|replicates", fmt.Sprintf("%d", a.NbootRep))

	_, jobs, err = p.galaxy.LaunchTool(tl)
	if err != nil {
		log.Print("Error while launching PhyML-SMS: " + err.Error())
		return
	}

	if len(jobs) != 1 {
		log.Print("Galaxy Error: No jobs in the list")
		err = errors.New("Galaxy error: No jobs in the list")
		return
	}
	a.JobId = jobs[0]
	p.db.UpdateAnalysis(a)
	return
}

func (p *GalaxyProcessor) submitFastTree(a *model.Analysis, alignfileid string) (err error) {
	var jobs []string

	tl := p.galaxy.NewToolLauncher(a.GalaxyHistory, p.fasttreeid)
	tl.AddFileInput("input_align", alignfileid, "hda")

	if a.AlignAlphabet == model.ALIGN_AMINOACIDS {
		tl.AddParameter("sequence_type|seqtype", "")
	} else if a.AlignAlphabet == model.ALIGN_NUCLEOTIDS {
		tl.AddParameter("sequence_type|seqtype", "-nt")
	} else {
		err = errors.New("Unkown sequence alphabet in alignment")
		return
	}
	tl.AddParameter("sequence_type|modelprot", "-lg")
	tl.AddParameter("sequence_type|modeldna", "-gtr")
	tl.AddParameter("gamma", "-gamma")
	tl.AddParameter("bootstrap|do_bootstrap", "true")
	tl.AddParameter("bootstrap|replicates", fmt.Sprintf("%d", a.NbootRep))

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

	a.JobId = jobs[0]
	p.db.UpdateAnalysis(a)
	return
}

func (p *GalaxyProcessor) submitToGalaxy(a *model.Analysis) (err error) {
	var reffileid string
	var bootfileid string
	var seqid string
	var history golaxy.HistoryFullInfo

	if p.stopping {
		err = errors.New("Booster server is stopping, please try again in a few minutes")
		log.Print("Error while submitting job : " + err.Error())
		return
	}

	// We create an history
	history, err = p.galaxy.CreateHistory("Booster History")
	if err != nil {
		log.Print("Error while Creating History: " + err.Error())
		return
	}
	log.Print("History: " + history.Id)
	a.GalaxyHistory = history.Id
	p.db.UpdateAnalysis(a)

	// If we have a sequence file, then we build the trees from it
	// and compute supports using the PHYML-SMS oneclick workflow from galaxy
	if a.SeqAlign != "" {

		if a.Workflow == model.WORKFLOW_NIL {
			err = errors.New("Phylogenetic workflow to launch is not defined")
			log.Print("Error while Uploading reference sequence file: " + err.Error())
			return
		}
		boostermem, boostercpu := estimateBoosterRunStats(a)
		if a.Workflow == model.WORKFLOW_PHYML_SMS {
			if mem, cpu := estimatePhyMLRunStats(a); (p.memlimit > 0 && math.Max(mem, boostermem) > float64(p.memlimit)) || (p.timeout > 0 && cpu+boostercpu > float64(p.timeout)) {
				err = errors.New("The given multiple alignment is too large to be analyzed online with PhyML-SMS, please consider using PhyML-SMS locally or using FastTree workflow")
				log.Print(fmt.Sprintf("%s: Tree: mem=%.2f,cpu=%2f; Booster: mem=%.2f,cpu=%2f", err.Error(), mem, cpu, boostermem, boostercpu))
				return
			}

			// The alignment was converted to phylip by server:newAnalysis function, now we upload it to history
			if seqid, _, err = p.galaxy.UploadFile(history.Id, a.SeqAlign, "phylip"); err != nil {
				log.Print("Error while Uploading reference sequence file: " + err.Error())
				return
			}
			if err = p.submitPhyML(a, seqid); err != nil {
				log.Print("Error while launching PhyML-SMS workflow : " + err.Error())
				return
			}
		} else if a.Workflow == model.WORKFLOW_FASTTREE {
			if mem, cpu := estimateFastTreeRunStats(a); (p.timeout > 0 && cpu+boostercpu > float64(p.timeout)) || (p.memlimit > 0 && math.Max(mem, boostermem) > float64(p.memlimit)) {
				err = errors.New("The given multiple alignment is too large to be analyzed online with FastTree, please consider using FastTree locally")
				log.Print(fmt.Sprintf("%s: Tree: mem=%.2f,cpu=%2f; Booster: mem=%.2f,cpu=%2f", err.Error(), mem, cpu, boostermem, boostercpu))
				return
			}

			// We upload the ref fasta sequence file to history
			if seqid, _, err = p.galaxy.UploadFile(history.Id, a.SeqAlign, "fasta"); err != nil {
				log.Print("Error while Uploading reference sequence file: " + err.Error())
				return
			}
			if err = p.submitFastTree(a, seqid); err != nil {
				log.Print("Error while launching FastTree workflow : " + err.Error())
				return
			}
		} else {
			err = errors.New("Error while launching workflow, unkown workflow")
			log.Print(err.Error())
			return
		}
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

		if err = p.submitBooster(a, reffileid, bootfileid); err != nil {
			log.Print("Error while launching Booster galaxy tool : " + err.Error())
			return
		}
	} else {
		log.Print("No Reference tree or Bootstrap tree given")
		err = errors.New("No Reference tree or Bootstrap tree given")
		return
	}

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
	// we delete the history
	if a.GalaxyHistory != "" {
		p.galaxy.DeleteHistory(a.GalaxyHistory)
	}
	// And delete the job from the running jobs
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
	p.stopping = true
	for _, a := range p.allRunningJobs() {
		log.Print("Cancelling job : " + a.Id)
		a.Status = model.STATUS_CANCELED
		a.End = time.Now().Format(time.RFC1123)
		a.Message = "Canceled after a server restart"
		if err = p.db.UpdateAnalysis(a); err != nil {
			log.Print(err)
		}
		p.rmRunningJob(a)
	}
	return
}

// Creates a new go routine that waits for
// new jobs in the queue and launches them on Galaxy
func (p *GalaxyProcessor) initJobLauncher() {
	go func() {
		for a := range p.queue {
			if p.stopping {
				break
			}
			log.Print(fmt.Sprintf("New analysis : id=%s", a.Id))
			err := p.submitToGalaxy(a)
			p.newRunningJob(a)
			if err != nil {
				log.Print("Error while submitting to galaxy: " + err.Error())
				a.Status = model.STATUS_ERROR
				a.End = time.Now().Format(time.RFC1123)
				a.Message = err.Error()
				p.rmRunningJob(a)
				if err = p.db.UpdateAnalysis(a); err != nil {
					log.Print("Problem updating job: " + err.Error())
				}
			}
		}
	}()
}

// Creates a new Go routine that monitors running jobs
func (p *GalaxyProcessor) initJobMonitor() {
	go func() {
		var state, fbptreeid, tbenormtreeid, tberawtreeid, tbelogid string
		var err error
		for !p.stopping {
			for _, job := range p.allRunningJobs() {
				state, fbptreeid, tbenormtreeid, tberawtreeid, tbelogid, err = p.checkJob(job)

				if state == "error" || job.Status == model.STATUS_ERROR {
					p.rmRunningJob(job)
					if err = p.notifier.Notify(job.StatusStr(), job.Id, job.RunName, job.WorkflowStr(), job.EMail); err != nil {
						log.Print(err)
					}
				} else if state == "ok" {
					if err = p.downloadResults(job, fbptreeid, tbenormtreeid, tberawtreeid, tbelogid); err != nil {
						job.Status = model.STATUS_ERROR
						job.Message = err.Error()
						log.Print(err)
					} else {
						job.Status = model.STATUS_FINISHED
						log.Print(fmt.Sprintf("Job %s finished successfully", job.Id))
					}
					p.rmRunningJob(job)
					if err = p.notifier.Notify(job.StatusStr(), job.Id, job.RunName, job.WorkflowStr(), job.EMail); err != nil {
						log.Print(err)
					}
				} else if t, _ := job.TimedOut(time.Duration(p.timeout) * time.Second); t {
					err = errors.New("Job timedout")
					job.Status = model.STATUS_TIMEOUT
					job.Message = "Time out: Job canceled"
					log.Print(fmt.Sprintf("Job %s timedout", job.Id))
					p.rmRunningJob(job)
					if err = p.notifier.Notify(job.StatusStr(), job.Id, job.RunName, job.WorkflowStr(), job.EMail); err != nil {
						log.Print(err)
					}
				} else if err != nil {
					log.Print(fmt.Sprintf("Error while checking job %s : %s", job.Id, err))
					time.Sleep(30 * time.Second)
					continue
				}

				if err = p.db.UpdateAnalysis(job); err != nil {
					log.Print(fmt.Sprintf("Problem updating job %s: %s", job.Id, err.Error()))
				}
				time.Sleep(1 * time.Second)
			}
			time.Sleep(10 * time.Second)
		}
	}()
}

func (p *GalaxyProcessor) restoreRunningJobs() {
	an, err := p.db.GetRunningAnalyses()
	if err != nil {
		log.Print(err.Error())
	} else {
		log.Print(fmt.Sprintf("Restoring %d Galaxy jobs", len(an)))
		for _, a := range an {
			p.newRunningJob(a)
		}
	}
}

func (p *GalaxyProcessor) downloadResults(a *model.Analysis, fbptreeid, tbenormtreeid, tberawtreeid, tbelogid string) (err error) {
	var outcontent []byte

	// We download resulting files
	if outcontent, err = p.galaxy.DownloadFile(a.GalaxyHistory, fbptreeid); err != nil {
		log.Print("Error while downloading fbp tree file: " + err.Error())
	}
	a.FbpTree = string(outcontent)

	// We scale branch supports from [0,nbootrep] to [0,1] for phyml
	if a.Workflow == model.WORKFLOW_PHYML_SMS {
		var t *tree.Tree
		if t, err = newick.NewParser(strings.NewReader(a.FbpTree)).Parse(); err != nil {
			log.Print("Error while scaling phyml branch supports to [0,1]: " + err.Error())
			return
		} else {
			t.ScaleSupports(1.0 / float64(a.NbootRep))
			a.FbpTree = t.Newick()
		}
	}

	if outcontent, err = p.galaxy.DownloadFile(a.GalaxyHistory, tbenormtreeid); err != nil {
		log.Print("Error while downloading support file: " + err.Error())
		return
	}
	a.TbeNormTree = string(outcontent)

	if outcontent, err = p.galaxy.DownloadFile(a.GalaxyHistory, tberawtreeid); err != nil {
		log.Print("Error while downloading avg dist tree file: " + err.Error())
		return
	}
	a.TbeRawTree = string(outcontent)

	if outcontent, err = p.galaxy.DownloadFile(a.GalaxyHistory, tbelogid); err != nil {
		log.Print("Error while downloading log file: " + err.Error())
		return
	}
	a.TbeLogs = cleanTBELogs(string(outcontent))
	return
}

func estimateFastTreeRunStats(a *model.Analysis) (mem, time float64) {
	alphabetsize := 4.0
	if a.AlignAlphabet == align.AMINOACIDS {
		alphabetsize = 20.0
	}

	time = 0.5071 +
		0.00000006141*math.Pow(float64(a.AlignNbSeq), 1.5)*math.Log(float64(a.AlignNbSeq))*float64(a.AlignLength)*alphabetsize
	mem = 2872 +
		0.003412*(math.Pow(float64(a.AlignNbSeq), 1.5)+float64(a.AlignNbSeq)*float64(a.AlignLength)*alphabetsize)
	time *= float64(a.NbootRep)
	return
}

func cleanTBELogs(log string) (cleanlog string) {
	ioregexp := regexp.MustCompile("(?m)^.*(Input|Output|Boot|Date|Seed|CPUs|End).*:.*$[\r\n]+")
	headregexp := regexp.MustCompile("(?m)^Taxon : tIndex$")
	titleregexp := regexp.MustCompile("(?m)^BOOSTER Support$[\r\n]+")
	cleanlog = ioregexp.ReplaceAllString(log, "")
	cleanlog = headregexp.ReplaceAllString(cleanlog, "Taxon : Instability")
	cleanlog = titleregexp.ReplaceAllString(cleanlog, "")
	return
}

func estimatePhyMLRunStats(a *model.Analysis) (mem, time float64) {
	alphabetweight := 0.0
	if a.AlignAlphabet == align.AMINOACIDS {
		alphabetweight = 1.1
	}

	time = 3.526 + 30.18*alphabetweight +
		0.00002227*float64(a.AlignNbSeq*a.AlignNbSeq*a.AlignLength) +
		0.00006672*alphabetweight*float64(a.AlignNbSeq*a.AlignNbSeq*a.AlignLength)
	mem = 3352.7636 -
		884.7005*alphabetweight +
		158.6359*float64(a.AlignNbSeq) -
		5.0467*float64(a.AlignLength) +
		81.0603*float64(a.AlignNbSeq)*alphabetweight -
		51.2838*float64(a.AlignLength)*alphabetweight +
		0.3754*float64(a.AlignLength*a.AlignNbSeq) +
		1.7922*float64(a.AlignLength*a.AlignNbSeq)*alphabetweight

	time *= float64(a.NbootRep)
	return
}

func estimateBoosterRunStats(a *model.Analysis) (mem, time float64) {
	time = math.Pow(-1.370621+
		0.002035*float64(a.AlignNbSeq), 2.0)
	mem = math.Pow(4865.453+
		9.197*float64(a.AlignNbSeq), 2)
	time *= float64(a.NbootRep)
	return
}
