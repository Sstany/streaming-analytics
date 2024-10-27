package db

const (
	initTableQuery = `CREATE TABLE IF NOT EXISTS jobs (
		id text PRIMARY KEY,
		state integer NOT NULL,
		source_url text NOT NULL,
		source_type integer NOT NULL,
	) WITHOUT ROWID;`
)

const (
	createJobQuery = `INSERT INTO jobs (id, state, source_url, source_type) VALUES ($1, $2, $3, $4)`
	readJobQuery   = `SELECT * FROM jobs WHERE jobs.id = $1`
)
