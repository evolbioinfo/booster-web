package database

import (
	"errors"
	"fmt"
	"log"

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
	id           string
	reffile      string
	bootfile     string
	result       string
	status       int
	message      string
	nboot        int
	startPending string
	startRunning string
	end          string
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
	rows, err := db.db.Query("SELECT * FROM analysis WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	dban := dbanalysis{}
	if rows.Next() {
		if err := rows.Scan(&dban.id, &dban.reffile, &dban.bootfile, &dban.result,
			&dban.status, &dban.message, &dban.nboot, &dban.startPending,
			&dban.startRunning, &dban.end); err != nil {
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
		dban.result,
		dban.status,
		model.StatusStr(dban.status),
		dban.message,
		dban.nboot,
		"",
		dban.startPending,
		dban.startRunning,
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
                    (id, reffile, bootfile, results, status, message, nboot, startpending, startrunning , end) 
                  VALUES (?,?,?,?,?,?,?,?,?,?) 
                  ON DUPLICATE KEY UPDATE results=values(results), status=values(status), 
                                          message=values(message), nboot=values(nboot), 
                                          startpending=values(startpending), startrunning=values(startrunning), end=values(end)`
	_, err := db.db.Exec(
		query,
		a.Id,
		a.Reffile,
		a.Bootfile,
		a.Result,
		a.Status,
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
		"CREATE TABLE if not exists analysis (id varchar(40) not null primary key, reffile blob, bootfile blob, results blob, status int, message blob, nboot int, startpending varchar(100), startrunning varchar(100), end varchar(100))")
	return err
}
