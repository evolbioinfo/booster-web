package processor

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/fredericlemoine/booster-web/database"
	"github.com/fredericlemoine/booster-web/model"
	"github.com/fredericlemoine/golaxy"
)

type GalaxyProcessor struct {
	runningJobs map[string]*model.Analysis
	galaxy      *golaxy.Galaxy
	queue       chan *model.Analysis // queue of analyses
	db          database.BoosterwebDB
	lock        sync.RWMutex
}

func (p *GalaxyProcessor) LaunchAnalysis(a *model.Analysis) (err error) {
	p.db.UpdateAnalysis(a)
	select {
	case p.queue <- a: // Put a in the channel unless it is full
	default:
		log.Print("Queue is full, cancelling job " + a.Id)
		//Channel full. Discarding value
		a.Status = model.STATUS_CANCELED
		a.StatusStr = model.StatusStr(a.Status)
		a.End = time.Now().Format(time.RFC1123)
		a.Message = "Computing queue is full, please try again in a few minutes"
		/* Insert analysis */
		err = p.db.UpdateAnalysis(a)
		a.DelTemp()
	}
	return
}

func (p *GalaxyProcessor) InitProcessor(url, apikey string, db database.BoosterwebDB, queuesize int) {
	p.db = db
	p.runningJobs = make(map[string]*model.Analysis)
	p.galaxy = golaxy.NewGalaxy(url, apikey)

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
					a.StatusStr = model.StatusStr(a.Status)
					a.Message = err.Error()
				}
				p.rmRunningJob(a)
				p.db.UpdateAnalysis(a)
			}
			log.Print(fmt.Sprintf("CPU %d : End", cpu))
		}(cpu)
	}
}

func (p *GalaxyProcessor) submitToGalaxy(a *model.Analysis) (err error) {
	var outcontent []byte
	var fileid string
	var fileid2 string
	var jobs []string
	var historyid string
	// We create an history
	historyid, err = p.galaxy.CreateHistory("Booster History")
	if err != nil {
		return
	}

	// We upload ref tree to history
	fileid, _, err = p.galaxy.UploadFile(historyid, a.Reffile)
	if err != nil {
		return
	}

	// We upload boot tree to history
	fileid2, _, err = p.galaxy.UploadFile(historyid, a.Bootfile)
	if err != nil {
		return
	}

	// We launch the job
	mapfiles := make(map[string]string)
	mapfiles["ref"] = fileid
	mapfiles["boot"] = fileid2
	_, jobs, err = p.galaxy.LaunchTool(historyid, "booster", mapfiles)
	if err != nil {
		return
	}

	if len(jobs) != 1 {
		err = errors.New("Galaxy error")
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
			id, ok := files["support"]
			if !ok {
				err = errors.New("Output file not present in the galaxy server")
				return
			}
			outcontent, err = p.galaxy.DownloadFile(historyid, id)
			if err != nil {
				return
			}
			log.Print(string(outcontent))
			a.Result = string(outcontent)
			a.Status = model.STATUS_FINISHED
			a.StatusStr = model.StatusStr(a.Status)
			a.End = time.Now().Format(time.RFC1123)
			a.Message = "Finished"
			err = p.db.UpdateAnalysis(a)
			return
		case "queued":
			a.Status = model.STATUS_PENDING
			a.StatusStr = model.StatusStr(a.Status)
			a.Message = "queued"
			err = p.db.UpdateAnalysis(a)
		case "waiting":
			a.Status = model.STATUS_PENDING
			a.StatusStr = model.StatusStr(a.Status)
			a.Message = "waiting"
			err = p.db.UpdateAnalysis(a)
		case "running":
			a.Status = model.STATUS_RUNNING
			if a.StartRunning == "" {
				a.StartRunning = time.Now().Format(time.RFC1123)
			}
			a.StatusStr = model.StatusStr(a.Status)
			a.Message = "running"
			err = p.db.UpdateAnalysis(a)
		case "new":
			a.Status = model.STATUS_PENDING
			a.StatusStr = model.StatusStr(a.Status)
			a.Message = "New Job"
			err = p.db.UpdateAnalysis(a)
		default:
			err = errors.New("Unkown Job state " + state)
			a.Status = model.STATUS_ERROR
			a.StatusStr = model.StatusStr(a.Status)
			log.Print("Job in unknown state " + state)
			err = p.db.UpdateAnalysis(a)
			return
		}

		time.Sleep(10 * time.Second)
	}
	return
}

/**
Keep a trace of currently running jobs
In order to cancel them when the server stops
*/
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