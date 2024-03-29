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

package database

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/evolbioinfo/booster-web/model"
)

type MemoryBoosterWebDB struct {
	lock        sync.RWMutex
	allanalyses map[string]*model.Analysis
}

/* Returns a new database */
func NewMemoryBoosterWebDB() *MemoryBoosterWebDB {
	log.Print("New in memory database")
	return &MemoryBoosterWebDB{}
}

func (db *MemoryBoosterWebDB) Connect() error {
	log.Print("Connecting in memory database")
	return nil
}

func (db *MemoryBoosterWebDB) Disconnect() error {
	log.Print("Disconnecting in memory database")
	return nil
}

func (db *MemoryBoosterWebDB) GetAnalysis(id string) (a *model.Analysis, err error) {
	db.lock.RLock()
	defer db.lock.RUnlock()
	var ok bool
	a, ok = db.allanalyses[id]
	if !ok {
		err = errors.New("Analysis does not exist")
	}
	return
}

/* Update an anlysis or insert it if it does not exist */
func (db *MemoryBoosterWebDB) UpdateAnalysis(a *model.Analysis) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	log.Print("In memory database : Insert or update analysis " + a.Id)
	db.allanalyses[a.Id] = a
	return nil
}

/* Check if table is present otherwise creates it */
func (db *MemoryBoosterWebDB) InitDatabase() error {
	log.Print("Initializing in memory database")
	db.allanalyses = make(map[string]*model.Analysis)
	return nil
}

// Will delete analyses older than d days
func (db *MemoryBoosterWebDB) DeleteOldAnalyses(days int) (err error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	log.Print("In memory database : Deleting old analyses")

	for id, a := range db.allanalyses {
		if o, _ := a.OlderThan(time.Duration(days*24) * time.Hour); o {
			delete(db.allanalyses, id)
		}
	}
	return

}

func (db *MemoryBoosterWebDB) GetRunningAnalyses() (analyses []*model.Analysis, err error) {
	analyses = make([]*model.Analysis, 0)
	return
}

func (db *MemoryBoosterWebDB) GetAnalysesPerDay() (perDay map[time.Time]int, err error) {
	perDay = make(map[time.Time]int)
	var toRound time.Time
	for _, a := range db.allanalyses {
		if toRound, err = time.Parse(time.RFC1123, a.StartPending); err != nil {
			return
		}
		rounded := time.Date(toRound.Year(), toRound.Month(), toRound.Day(), 0, 0, 0, 0, toRound.Location())
		perDay[rounded] = perDay[rounded] + 1
	}
	return
}

func (db *MemoryBoosterWebDB) GetAnalysesStats() (pendingJobs, runningJobs, finishedJobs, canceledJobs, errorJobs, timeoutJobs int, avgJobsPerDay float64, err error) {
	var toRound time.Time

	minDay := time.Now()
	maxDay := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	total := 0.0
	for _, a := range db.allanalyses {
		if toRound, err = time.Parse(time.RFC1123, a.StartPending); err != nil {
			return
		}
		rounded := time.Date(toRound.Year(), toRound.Month(), toRound.Day(), 0, 0, 0, 0, toRound.Location())
		total++
		if rounded.Before(minDay) {
			minDay = rounded
		}
		if rounded.After(maxDay) {
			maxDay = rounded
		}

		switch a.Status {
		case model.STATUS_PENDING:
			pendingJobs++
		case model.STATUS_RUNNING:
			runningJobs++
		case model.STATUS_FINISHED:
			finishedJobs++
		case model.STATUS_ERROR:
			errorJobs++
		case model.STATUS_CANCELED:
			canceledJobs++
		case model.STATUS_TIMEOUT:
			timeoutJobs++
		case model.STATUS_DELETED:
			finishedJobs++
		default:
		}
	}

	avgJobsPerDay = total / float64(int(maxDay.Sub(minDay).Hours()/24))

	return
}
