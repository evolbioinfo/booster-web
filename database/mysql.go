package database

import (
	"errors"
	"fmt"

	"database/sql"
	"github.com/fredericlemoine/booster-web/model"
	_ "github.com/go-sql-driver/mysql"
)

type MySQLBoosterwebDB struct {
	login  string
	pass   string
	url    string
	dbname string
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
func NewMySQLBoosterwebDB(login, pass, url, dbname string) *MySQLBoosterwebDB {
	return &MySQLBoosterwebDB{
		login,
		pass,
		url,
		dbname,
		nil,
	}
}

func (db *MySQLBoosterwebDB) Connect() error {
	d, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s", db.login, db.pass, db.url, db.dbname))
	if err != nil {
		db.db = d
	}
	return err
}

func (db *MySQLBoosterwebDB) Disconnect() error {
	if db.db == nil {
		return errors.New("Database not opened")
	}
	return db.db.Close()
}

func (db *MySQLBoosterwebDB) GetAnalysis(id string) (*model.Analysis, error) {
	if db.db == nil {
		return nil, errors.New("Database not opened")
	}
	rows, err := db.db.Query("SELECT * FROM analysis WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	dban := &dbanalysis{}
	if rows.Next() {
		if err := rows.Scan(&dban); err != nil {
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
	if db.db == nil {
		return errors.New("Database not opened")
	}
	query := "INSERT INTO analysis (id, reffile, bootfile, results, status, message, nboot, startpending, startrunning , end) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON DUPLICATE KEY UPDATE"
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
	_, err := db.db.Exec(
		"CREATE TABLE analysis if not exists (id varchar(40) not null primary key, reffile blob, bootfile blob, results blob, status int, message blob, nboot int, startpending varchar(100), startrunning varchar(100), end varchar(100))")
	return err
}
