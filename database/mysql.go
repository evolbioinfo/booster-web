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
	"strings"

	"database/sql"
	"github.com/fredericlemoine/booster-web/model"
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
	id           string `mysql-type:varchar(100)` // Id of the analysis
	email        string `mysql-type:varchar(100)` // Email of the analysis creator
	seqfile      string `mysql-type:blob`         // Input Fasta Sequence File if user wants to build the ref/boot trees (priority over reffile and bootfile)
	nbootrep     int    `mysql-type:int`          // Number of bootstrap replicates given by the user to build the bootstrap trees
	alignfile    string `mysql-type:blob`         // alignment input file (if user wants to build the trees)
	workflow     int    `mysql-type:int`          // workflow to launch if alignfile!="" : 8: PhyML-SMS, 9: FastTRee
	reffile      string `mysql-type:blob`         // reference tree file
	bootfile     string `mysql-type:blob`         // boot tree file
	results      string `mysql-type:longtext`     // result tree with normalize supports
	rawtree      string `mysql-type:longtext`     // result tree with raw <id|avg_dist|depth> as branch names
	reslogs      string `mysql-type:longtext`     // log file
	status       int    `mysql-type:int`          // Status of the analysis
	algorithm    int    `mysql-type:int`          // Algorithm used
	message      string `mysql-type:message`      // Optional message
	nboot        int    `mysql-type:int`          // number of bootstrap trees
	startpending string `mysql-type:varchar(100)` // date of job being submited
	startrunning string `mysql-type:varchar(100)` // date of job being running
	end          string `mysql-type:varchar(100)` // date of job finished
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
	rows, err := db.db.Query("SELECT id,email,seqfile,nbootrep,alignfile,workflow,reffile,bootfile,results,rawtree,reslogs,status,algorithm,message,nboot,startpending,startrunning,end FROM analysis WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dban := dbanalysis{}
	if rows.Next() {
		if err := rows.Scan(&dban.id, &dban.email, &dban.seqfile, &dban.nbootrep,
			&dban.alignfile, &dban.workflow, &dban.reffile, &dban.bootfile,
			&dban.results, &dban.rawtree, &dban.reslogs, &dban.status, &dban.algorithm,
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
		Id:           dban.id,
		EMail:        dban.email,
		SeqFile:      dban.seqfile,
		NbootRep:     dban.nbootrep,
		Alignfile:    dban.alignfile,
		Workflow:     dban.workflow,
		Reffile:      dban.reffile,
		Bootfile:     dban.bootfile,
		Result:       dban.results,
		RawTree:      dban.rawtree,
		ResLogs:      dban.reslogs,
		Status:       dban.status,
		Algorithm:    dban.algorithm,
		StatusStr:    model.StatusStr(dban.status),
		Message:      dban.message,
		Nboot:        dban.nboot,
		Collapsed:    "",
		StartPending: dban.startpending,
		StartRunning: dban.startrunning,
		End:          dban.end,
	}

	return a, nil
}

/* Update an anlysis or insert it if it does not exist */
func (db *MySQLBoosterwebDB) UpdateAnalysis(a *model.Analysis) error {
	log.Print("Mysql database : Insert or update analysis " + a.Id)

	if db.db == nil {
		return errors.New("Database not opened")
	}
	query := `INSERT INTO analysis 
                    (id, email, seqfile, nbootrep, alignfile, workflow, reffile, bootfile, results, rawtree, reslogs, status, algorithm, message, nboot, startpending, startrunning , end) 
                  VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) 
                  ON DUPLICATE KEY UPDATE alignfile=values(alignfile), results=values(results), rawtree=values(rawtree), reslogs=values(reslogs), 
                                          status=values(status), algorithm=values(algorithm),workflow=values(workflow), message=values(message), 
                                          nboot=values(nboot),startpending=values(startpending), startrunning=values(startrunning), end=values(end)`
	_, err := db.db.Exec(
		query,
		a.Id,
		a.EMail,
		a.SeqFile,
		a.NbootRep,
		a.Alignfile,
		a.Workflow,
		a.Reffile,
		a.Bootfile,
		a.Result,
		a.RawTree,
		a.ResLogs,
		a.Status,
		a.Algorithm,
		a.Message,
		a.Nboot,
		a.StartPending,
		a.StartRunning,
		a.End,
	)
	return err
}

/* Check if table is present otherwise creates it */
func (db *MySQLBoosterwebDB) InitDatabase() error {
	log.Print("Initializing mysql Database")
	_, err := db.db.Exec(
		"CREATE TABLE if not exists analysis (id varchar(100) not null primary key, email varchar(100), seqfile blob, nbootrep int, alignfile blob, workflow int, reffile blob, bootfile blob, results longtext, rawtree longtext, reslogs longtext, status int, algorithm int, message blob, nboot int, startpending varchar(100), startrunning varchar(100), end varchar(100))")

	if err == nil {
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
		if strings.HasPrefix(string(field.Tag), "mysql-type:") {
			_, ok := cols[field.Name]
			if !ok {
				log.Print(fmt.Sprintf("Adding database column %s", field.Name))

				tabs := strings.Split(string(field.Tag), ":")
				if len(tabs) == 2 {
					_, err = db.db.Exec("ALTER TABLE analysis ADD COLUMN " + field.Name + " " + tabs[1])
					if err != nil {
						return err
					}
				} else {
					return errors.New(fmt.Sprintf("dbanalysis struct element %s does not have proper mysql type: '%s'", field.Name, string(field.Tag)))
				}
			}
		} else {
			return errors.New(fmt.Sprintf("dbanalysis struct element %s does not have mysql type", field.Name))
		}
	}

	return err
}
