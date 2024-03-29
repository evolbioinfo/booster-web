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
	"fmt"
	"log"
	"reflect"
	"time"

	"database/sql"

	"github.com/evolbioinfo/booster-web/model"
	_ "github.com/go-sql-driver/mysql"
)

type MySQLBoosterwebDB struct {
	login  string
	pass   string
	url    string
	dbname string
	port   int
	db     *sql.DB
}

type dbanalysis struct {
	id            string `mysql-type:"varchar(100)" mysql-other:"NOT NULL PRIMARY KEY"` // Id of the analysis
	runname       string `mysql-type:"varchar(100)" mysql-default:"''"`                 // Optional user given name of the run
	email         string `mysql-type:"varchar(100)" mysql-default:"''"`                 // Email of the analysis creator
	seqalign      string `mysql-type:"blob"`                                            // Input Fasta Sequence Alignment if user wants to build the ref/boot trees (priority over reffile and bootfile)
	nbootrep      int    `mysql-type:"int" mysql-default:"0"`                           // Number of bootstrap replicates given by the user to build the bootstrap trees
	alignfile     string `mysql-type:"longblob"`                                        // alignment input file (if user wants to build the trees)
	alignalphabet int    `mysql-type:"int" mysql-default:"-1"`                          // alignment alphabet 0: aa | 1: nt
	workflow      int    `mysql-type:"int" mysql-default:"-1"`                          // workflow to launch if alignfile!="" : 8: PhyML-SMS, 9: FastTRee
	alignnbseq    int    `mysql-type:"int" mysql-default:"-1"`                          // Number of sequences in the given alignment
	alignlength   int    `mysql-type:"int" mysql-default:"-1"`                          // Length of the given alignment
	reffile       string `mysql-type:"blob"`                                            // reference tree file
	bootfile      string `mysql-type:"blob"`                                            // boot tree file
	fbptree       string `mysql-type:"longtext"`                                        // tree with fbp supports
	tbenormtree   string `mysql-type:"longtext"`                                        // tree with normalized tbe supports
	tberawtree    string `mysql-type:"longtext"`                                        // tree with raw tbe supports in the form <id|avg_dist|depth> as branch names
	tbelogs       string `mysql-type:"longtext"`                                        // tbe log file
	status        int    `mysql-type:"int" mysql-default:"-1"`                          // Status of the analysis
	jobid         string `mysql-type:"varchar(100)" mysql-default:"''"`                 // Galaxy or local Job id
	galaxyhistory string `mysql-type:"varchar(100)" mysql-default:"''"`                 // Galaxy History
	message       string `mysql-type:"longtext"`                                        // Optional message
	nboot         int    `mysql-type:"int" mysql-default:"0"`                           // number of bootstrap trees
	startpending  string `mysql-type:"varchar(100)" mysql-default:"''"`                 // date of job being submited
	startrunning  string `mysql-type:"varchar(100)" mysql-default:"''"`                 // date of job being running
	end           string `mysql-type:"varchar(100)" mysql-default:"''"`                 // date of job finished
}

/* Returns a new database */
func NewMySQLBoosterwebDB(login, pass, url, dbname string, port int) *MySQLBoosterwebDB {
	log.Print("New mysql database")
	return &MySQLBoosterwebDB{
		login,
		pass,
		url,
		dbname,
		port,
		nil,
	}
}

func (db *MySQLBoosterwebDB) Connect() error {
	log.Print("Connect mysql database")
	d, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", db.login, db.pass, db.url, db.port, db.dbname))
	if err != nil {
		log.Print(err)
	} else {
		db.db = d
	}
	return err
}

func (db *MySQLBoosterwebDB) Disconnect() error {
	log.Print("Disconnect mysql database")
	if db.db == nil {
		return errors.New("Database not opened")
	}
	return db.db.Close()
}

func (db *MySQLBoosterwebDB) GetAnalysis(id string) (*model.Analysis, error) {
	if db.db == nil {
		return nil, errors.New("Database not opened")
	}
	rows, err := db.db.Query("SELECT id,runname,email,seqalign,nbootrep,alignfile,alignalphabet,workflow,alignnbseq,alignlength,reffile,bootfile,fbptree,tbenormtree,tberawtree,tbelogs,status,jobid,galaxyhistory,message,nboot,startpending,startrunning,end FROM analysis WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dban := dbanalysis{}
	if rows.Next() {
		if err := rows.Scan(&dban.id, &dban.runname, &dban.email, &dban.seqalign, &dban.nbootrep,
			&dban.alignfile, &dban.alignalphabet, &dban.workflow, &dban.alignnbseq, &dban.alignlength, &dban.reffile, &dban.bootfile,
			&dban.fbptree, &dban.tbenormtree, &dban.tberawtree, &dban.tbelogs, &dban.status, &dban.jobid, &dban.galaxyhistory,
			&dban.message, &dban.nboot, &dban.startpending, &dban.startrunning, &dban.end); err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("Analysis does not exist")
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	a := &model.Analysis{
		Id:            dban.id,
		RunName:       dban.runname,
		EMail:         dban.email,
		SeqAlign:      dban.seqalign,
		NbootRep:      dban.nbootrep,
		Alignfile:     dban.alignfile,
		AlignAlphabet: dban.alignalphabet,
		Workflow:      dban.workflow,
		AlignNbSeq:    dban.alignnbseq,
		AlignLength:   dban.alignlength,
		Reffile:       dban.reffile,
		Bootfile:      dban.bootfile,
		FbpTree:       dban.fbptree,
		TbeNormTree:   dban.tbenormtree,
		TbeRawTree:    dban.tberawtree,
		TbeLogs:       dban.tbelogs,
		Status:        dban.status,
		JobId:         dban.jobid,
		GalaxyHistory: dban.galaxyhistory,
		Message:       dban.message,
		Nboot:         dban.nboot,
		StartPending:  dban.startpending,
		StartRunning:  dban.startrunning,
		End:           dban.end,
	}

	return a, nil
}

// Get only analyses that are running (1) or pending (0)
func (db *MySQLBoosterwebDB) GetRunningAnalyses() (analyses []*model.Analysis, err error) {
	if db.db == nil {
		return nil, errors.New("Database not opened")
	}
	analyses = make([]*model.Analysis, 0)
	var rows *sql.Rows
	query := `SELECT id,runname, email,seqalign,nbootrep,alignfile,
                         alignalphabet,workflow,alignnbseq,alignlength,reffile,bootfile,
                         fbptree,tbenormtree,tberawtree,tbelogs,status,jobid,galaxyhistory,
                         message,nboot,startpending,startrunning,end 
                  FROM analysis 
                  WHERE status=0 or status=1`
	if rows, err = db.db.Query(query); err != nil {
		return
	}
	defer rows.Close()

	dban := dbanalysis{}
	for rows.Next() {
		if err = rows.Scan(&dban.id, &dban.runname, &dban.email, &dban.seqalign, &dban.nbootrep,
			&dban.alignfile, &dban.alignalphabet, &dban.workflow, &dban.alignnbseq, &dban.alignlength, &dban.reffile, &dban.bootfile,
			&dban.fbptree, &dban.tbenormtree, &dban.tberawtree, &dban.tbelogs, &dban.status, &dban.jobid, &dban.galaxyhistory,
			&dban.message, &dban.nboot, &dban.startpending, &dban.startrunning, &dban.end); err != nil {
			return
		}
		if err = rows.Err(); err != nil {
			return
		}

		a := &model.Analysis{
			Id:            dban.id,
			RunName:       dban.runname,
			EMail:         dban.email,
			SeqAlign:      dban.seqalign,
			NbootRep:      dban.nbootrep,
			Alignfile:     dban.alignfile,
			AlignAlphabet: dban.alignalphabet,
			Workflow:      dban.workflow,
			AlignNbSeq:    dban.alignnbseq,
			AlignLength:   dban.alignlength,
			Reffile:       dban.reffile,
			Bootfile:      dban.bootfile,
			FbpTree:       dban.fbptree,
			TbeNormTree:   dban.tbenormtree,
			TbeRawTree:    dban.tberawtree,
			TbeLogs:       dban.tbelogs,
			Status:        dban.status,
			JobId:         dban.jobid,
			GalaxyHistory: dban.galaxyhistory,
			Message:       dban.message,
			Nboot:         dban.nboot,
			StartPending:  dban.startpending,
			StartRunning:  dban.startrunning,
			End:           dban.end,
		}
		analyses = append(analyses, a)
	}

	return
}

/* Update an anlysis or insert it if it does not exist */
func (db *MySQLBoosterwebDB) UpdateAnalysis(a *model.Analysis) error {
	//log.Print("Mysql database : Insert or update analysis " + a.Id)

	if db.db == nil {
		return errors.New("Database not opened")
	}
	query := `INSERT INTO analysis 
                    (id, runname, email, seqalign, nbootrep, alignfile, alignalphabet,workflow, alignnbseq, alignlength, reffile, bootfile, fbptree,tbenormtree, tberawtree, tbelogs, status, jobid, galaxyhistory, message, nboot, startpending, startrunning , end) 
                  VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) 
                  ON DUPLICATE KEY UPDATE runname=values(runname), alignfile=values(alignfile),alignalphabet=values(alignalphabet),fbptree=values(fbptree), 
                                          tbenormtree=values(tbenormtree), tberawtree=values(tberawtree), tbelogs=values(tbelogs), 
                                          status=values(status),jobid=values(jobid),galaxyhistory=values(galaxyhistory),workflow=values(workflow), 
                                          alignnbseq=values(alignnbseq), alignLength=values(alignLength), message=values(message), nboot=values(nboot),
                                          startpending=values(startpending), startrunning=values(startrunning), end=values(end)`
	_, err := db.db.Exec(
		query,
		a.Id,
		a.RunName,
		a.EMail,
		a.SeqAlign,
		a.NbootRep,
		a.Alignfile,
		a.AlignAlphabet,
		a.Workflow,
		a.AlignNbSeq,
		a.AlignLength,
		a.Reffile,
		a.Bootfile,
		a.FbpTree,
		a.TbeNormTree,
		a.TbeRawTree,
		a.TbeLogs,
		a.Status,
		a.JobId,
		a.GalaxyHistory,
		a.Message,
		a.Nboot,
		a.StartPending,
		a.StartRunning,
		a.End,
	)
	return err
}

/* Check if table is present otherwise creates it */
func (db *MySQLBoosterwebDB) InitDatabase() (err error) {
	log.Print("Initializing mysql Database")
	query := "CREATE TABLE if not exists analysis ("
	dba := dbanalysis{}
	dbanalysistype := reflect.ValueOf(dba).Type()
	fields := dbanalysistype.NumField()
	for i := 0; i < fields; i++ {
		field := dbanalysistype.Field(i)
		if mysqltype, mysqltypeok := field.Tag.Lookup("mysql-type"); mysqltypeok {
			mysqldefault, mysqldefaultok := field.Tag.Lookup("mysql-default")
			mysqlother, mysqlotherok := field.Tag.Lookup("mysql-other")
			if i > 0 {
				query += ","
			}
			query += field.Name + " " + mysqltype
			if mysqlotherok {
				query += " " + mysqlother
			}
			// If there is a default value for this field
			if mysqldefaultok {
				query += " DEFAULT " + mysqldefault
			}
		} else {
			return errors.New(fmt.Sprintf("Cannot create table, dbanalysis struct element %s does not have mysql type", field.Name))
		}
	}
	query += ");"
	if _, err = db.db.Exec(query); err == nil {
		err = db.checkColumns()
	}
	return err
}

/* Check if table has all the columns, otherwise adds them */
func (db *MySQLBoosterwebDB) checkColumns() error {
	log.Print("Checking database tables")

	rows, err := db.db.Query("SELECT * FROM analysis")
	cols := make(map[string]bool)

	if colnames, err := rows.Columns(); err == nil {
		for _, col := range colnames {
			cols[col] = true
		}
	} else {
		return err
	}

	dba := dbanalysis{}
	dbanalysistype := reflect.ValueOf(dba).Type()
	fields := dbanalysistype.NumField()
	for i := 0; i < fields; i++ {
		field := dbanalysistype.Field(i)
		if mysqltype, mysqltypeok := field.Tag.Lookup("mysql-type"); mysqltypeok {
			mysqldefault, mysqldefaultok := field.Tag.Lookup("mysql-default")
			mysqlother, mysqlotherok := field.Tag.Lookup("mysql-other")
			if _, colok := cols[field.Name]; !colok {
				log.Print(fmt.Sprintf("Adding database column %s", field.Name))
				query := "ALTER TABLE analysis ADD COLUMN " + field.Name + " " + mysqltype
				// If there is a default value for this field
				if mysqldefaultok {
					query += " DEFAULT " + mysqldefault
				}
				if mysqlotherok {
					query += " " + mysqlother
				}
				_, err = db.db.Exec(query)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New(fmt.Sprintf("dbanalysis struct element %s does not have mysql type", field.Name))
		}
	}

	return err
}

// Will delete analyses older than d days
func (db *MySQLBoosterwebDB) DeleteOldAnalyses(days int) (err error) {
	log.Print("Mysql database : Deleting old analyses")
	if db.db == nil {
		return errors.New("Database not opened")
	}

	query := `UPDATE analysis set alignfile='',fbptree='', tbenormtree='', tberawtree='',tbelogs='',status=6 where status<>0 and status<>1 and STR_TO_DATE(replace(replace(end,"CET",""),"CEST",""), '%a, %d %b %Y %H:%i:%S')<DATE_SUB(CURDATE(), INTERVAL `
	query += fmt.Sprintf("%d", days) + " DAY);"

	_, err = db.db.Exec(query)

	return
}
func (db *MySQLBoosterwebDB) GetAnalysesPerDay() (perDay map[time.Time]int, err error) {
	perDay = make(map[time.Time]int)

	if db.db == nil {
		err = errors.New("Database not opened")
		return
	}
	var rows *sql.Rows
	query := `SELECT startpending FROM analysis`

	if rows, err = db.db.Query(query); err != nil {
		return
	}
	defer rows.Close()

	var start string
	var toRound time.Time
	for rows.Next() {
		if err = rows.Scan(&start); err != nil {
			return
		}
		if err = rows.Err(); err != nil {
			return
		}

		if toRound, err = time.Parse(time.RFC1123, start); err != nil {
			return
		}

		rounded := time.Date(toRound.Year(), toRound.Month(), toRound.Day(), 0, 0, 0, 0, toRound.Location())
		perDay[rounded] = perDay[rounded] + 1
	}
	return
}

func (db *MySQLBoosterwebDB) GetAnalysesStats() (pendingJobs, runningJobs, finishedJobs, canceledJobs, errorJobs, timeoutJobs int, avgJobsPerDay float64, err error) {
	minDay := time.Now()
	maxDay := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	total := 0.0

	if db.db == nil {
		err = errors.New("Database not opened")
		return
	}
	var rows *sql.Rows
	query := `SELECT startpending,status FROM analysis`

	if rows, err = db.db.Query(query); err != nil {
		return
	}
	defer rows.Close()

	var start string
	var status int
	var toRound time.Time
	for rows.Next() {
		if err = rows.Scan(&start, &status); err != nil {
			return
		}
		if err = rows.Err(); err != nil {
			return
		}

		if toRound, err = time.Parse(time.RFC1123, start); err != nil {
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

		switch status {
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
