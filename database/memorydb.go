package database

import (
	"errors"
	"log"
	"sync"

	"github.com/fredericlemoine/booster-web/model"
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
