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
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/fredericlemoine/booster-web/database"
	"github.com/fredericlemoine/booster-web/io"
	"github.com/fredericlemoine/booster-web/model"
	"github.com/fredericlemoine/booster-web/notification"
	"github.com/fredericlemoine/gotree/io/utils"
	"github.com/fredericlemoine/gotree/support"
	"github.com/fredericlemoine/gotree/tree"
)

const (
	RUNNERS_QUEUESIZE_DEFAULT  = 10
	RUNNERS_NBRUNNERS_DEFAULT  = 1
	RUNNERS_TIMEOUT_DEFAULT    = 0 // unlimited
	RUNNERS_JOBTHREADS_DEFAULT = 1
)

type LocalProcessor struct {
	runningJobs map[string]*model.Analysis
	queue       chan *model.Analysis // queue of analyses
	db          database.BoosterwebDB
	notifier    notification.Notifier
	lock        sync.RWMutex
}

func (p *LocalProcessor) LaunchAnalysis(a *model.Analysis) (err error) {
	if a.SeqFile != "" {
		err = errors.New("Local processor cannot infer trees, sequence file won't be analyzed")
		a.DelTemp()
		return
	}
	select {
	case p.queue <- a: // Put a in the channel unless it is full
	default:
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

func (p *LocalProcessor) InitProcessor(nbrunners, queuesize, timeout, jobthreads int, db database.BoosterwebDB, notifier notification.Notifier) {
	var maxcpus int = runtime.NumCPU() // max number of cpus

	p.db = db
	p.notifier = notifier
	p.runningJobs = make(map[string]*model.Analysis)

	if jobthreads == 0 {
		jobthreads = RUNNERS_JOBTHREADS_DEFAULT
	}

	if nbrunners == 0 {
		nbrunners = RUNNERS_NBRUNNERS_DEFAULT
	}
	if (nbrunners*jobthreads + 1) > maxcpus {
		log.Fatal(fmt.Sprintf("Your system does not have enough cpus to run the http server + %d bootstrap runners with each %d threads", nbrunners, jobthreads))
	}

	if queuesize == 0 {
		queuesize = RUNNERS_QUEUESIZE_DEFAULT
	}
	if queuesize <= 0 {
		log.Fatal("The queue size must be set to a value >0")
	}
	if queuesize < 100 {
		log.Print("The queue size is <100, it may be a problem for users")
	}
	log.Print("Init local processor")
	log.Print(fmt.Sprintf("Nb runners: %d", nbrunners))
	log.Print(fmt.Sprintf("Queue size: %d", queuesize))
	log.Print(fmt.Sprintf("Job timeout: %ds", timeout))
	log.Print(fmt.Sprintf("Job threads: %d", jobthreads))

	p.queue = make(chan *model.Analysis, queuesize)

	// We initialize computing routines
	for cpu := 0; cpu < nbrunners; cpu++ {
		go func(cpu int) {
			var tbesupporter support.Supporter
			var fbpsupporter support.Supporter

			for a := range p.queue {
				log.Print(fmt.Sprintf("CPU=%d | New analysis, id=%s", cpu, a.Id))

				a.Status = model.STATUS_RUNNING
				a.StartRunning = time.Now().Format(time.RFC1123)

				io.LogInfo("Booster supporter")
				tbesupporter = support.NewBoosterSupporter(true, true, false, true, 0.3, false)
				io.LogInfo("Classical supporter")
				fbpsupporter = support.NewClassicalSupporter(true)
				finished := false
				er := p.db.UpdateAnalysis(a)
				if er != nil {
					io.LogError(er)
					continue
				}
				p.newRunningJob(a)
				var wg sync.WaitGroup // For waiting end of step computation
				wg.Add(1)
				go func() {
					var err error
					if err = p.computeSupport(tbesupporter, a, jobthreads, true); err != nil {
						io.LogError(err)
						a.Message = err.Error()
						a.Status = model.STATUS_ERROR
						return
					}
					if err = p.computeSupport(fbpsupporter, a, jobthreads, false); err != nil {
						io.LogError(err)
						a.Message = err.Error()
						a.Status = model.STATUS_ERROR
						return
					}

					if err = p.db.UpdateAnalysis(a); err != nil {
						io.LogError(er)
					}

					p.rmRunningJob(a)

					a.DelTemp()
					if err = p.notifier.Notify(a.StatusStr(), a.Id, a.EMail); err != nil {
						io.LogError(err)
					}
					wg.Done()
				}()

				go func() {
					for {
						a.Nboot = tbesupporter.Progress()
						p.db.UpdateAnalysis(a)
						if finished {
							break
						}
						time.Sleep(4 * time.Second)
					}
				}()

				if timeout > 0 {
					go func() {
						time.Sleep(time.Duration(timeout) * time.Second)
						if !finished {
							tbesupporter.Cancel()
							fbpsupporter.Cancel()
						}
					}()
				}
				wg.Wait()
				a.Nboot = tbesupporter.Progress()
				p.db.UpdateAnalysis(a)
				finished = true
			}
			log.Print(fmt.Sprintf("CPU %d : End", cpu))
		}(cpu)
	}
}

/**
Keep a trace of currently running jobs
In order to cancel them when the server stops
*/
func (p *LocalProcessor) newRunningJob(a *model.Analysis) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.runningJobs[a.Id] = a
}

func (p *LocalProcessor) rmRunningJob(a *model.Analysis) {
	p.lock.Lock()
	defer p.lock.Unlock()

	delete(p.runningJobs, a.Id)
}

func (p *LocalProcessor) allRunningJobs() []*model.Analysis {
	p.lock.RLock()
	defer p.lock.RUnlock()
	v := make([]*model.Analysis, 0)
	for _, value := range p.runningJobs {
		v = append(v, value)
	}
	return v
}

func (p *LocalProcessor) CancelAnalyses() (err error) {
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

func (p *LocalProcessor) computeSupport(supporter support.Supporter, a *model.Analysis, jobThreads int, tbe bool) (err error) {
	var refTree *tree.Tree
	var treeFile, tmpFile *os.File
	var treeReader *bufio.Reader
	var treeChannel <-chan tree.Trees
	var dat []byte

	if refTree, err = utils.ReadTree(a.Reffile, utils.FORMAT_NEWICK); err != nil {
		io.LogError(err)
		return
	}

	treeFile, treeReader, err = utils.GetReader(a.Bootfile)
	defer treeFile.Close()
	if err != nil {
		io.LogError(err)
		return
	}

	treeChannel = utils.ReadMultiTrees(treeReader, utils.FORMAT_NEWICK)
	tmpFile, err = ioutil.TempFile("", "booster_log")
	defer os.Remove(tmpFile.Name()) // clean up
	if err != nil {
		io.LogError(err)
		return
	}

	err = support.ComputeSupport(refTree, treeChannel, tmpFile, jobThreads, supporter)
	a.End = time.Now().Format(time.RFC1123)
	tmpFile.Close()
	if err != nil {
		return
		io.LogError(err)
		return
	}

	if supporter.Canceled() {
		a.Status = model.STATUS_TIMEOUT
	} else {
		a.Status = model.STATUS_FINISHED
	}
	refTree.ClearPvalues()

	if tbe {
		// We  print the raw support tree first
		reformated := refTree.Clone()
		support.ReformatAvgDistance(reformated)
		a.TbeRawTree = reformated.Newick()
		// We normalize the supports and save the tree
		support.NormalizeTransferDistancesByDepth(refTree)

		if dat, err = ioutil.ReadFile(tmpFile.Name()); err != nil {
			io.LogError(err)
			return
		}

		a.TbeLogs = string(dat)
		a.TbeNormTree = refTree.Newick()
		a.Message = "Finished TBE"
	} else {
		a.FbpTree = refTree.Newick()
		a.Message = "Finished FBP"
	}

	return
}
