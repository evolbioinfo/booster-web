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
	id           string `mysql-type:varchar(100)`
	reffile      string `mysql-type:blob`
	bootfile     string `mysql-type:blob`
	results      string `mysql-type:longtext`
	status       int    `mysql-type:int`
	algorithm    int    `mysql-type:int`
	message      string `mysql-type:message`
	nboot        int    `mysql-type:int`
	startpending string `mysql-type:varchar(100)`
	startrunning string `mysql-type:varchar(100)`
	end          string `mysql-type:varchar(100)`
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
	rows, err := db.db.Query("SELECT id,reffile,bootfile,results,status,algorithm,message,nboot,startpending,startrunning,end FROM analysis WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dban := dbanalysis{}
	if rows.Next() {
		if err := rows.Scan(&dban.id, &dban.reffile, &dban.bootfile, &dban.results,
			&dban.status, &dban.algorithm, &dban.message, &dban.nboot, &dban.startpending,
			&dban.startrunning, &dban.end); err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("Analysis does not exist")
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	a := &model.Analysis{
		dban.id,
		dban.reffile,
		dban.bootfile,
		dban.results,
		dban.status,
		dban.algorithm,
		model.StatusStr(dban.status),
		dban.message,
		dban.nboot,
		"",
		dban.startpending,
		dban.startrunning,
		dban.end,
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
                    (id, reffile, bootfile, results, status, algorithm, message, nboot, startpending, startrunning , end) 
                  VALUES (?,?,?,?,?,?,?,?,?,?,?) 
                  ON DUPLICATE KEY UPDATE results=values(results), status=values(status), algorithm=values(algorithm),
                                          message=values(message), nboot=values(nboot), 
                                          startpending=values(startpending), startrunning=values(startrunning), end=values(end)`
	_, err := db.db.Exec(
		query,
		a.Id,
		a.Reffile,
		a.Bootfile,
		a.Result,
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
		"CREATE TABLE if not exists analysis (id varchar(40) not null primary key, reffile blob, bootfile blob, results longtext, status int, algorithm int, message blob, nboot int, startpending varchar(100), startrunning varchar(100), end varchar(100))")

	if err == nil {
		err = db.checkColumns()
	}
	return err
}

/* Check if table has all the columns, otherwize adds them */
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
