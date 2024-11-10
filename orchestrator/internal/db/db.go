package db

const (
	initTableQuery = `CREATE TABLE IF NOT EXISTS jobs (
		id text PRIMARY KEY,
		state integer NOT NULL,
		source_url text NOT NULL
	)`
)

const (
	createJobQuery = `INSERT INTO jobs (id, state, source_url) VALUES ($1, $2, $3)`
	readJobQuery   = `SELECT * FROM jobs WHERE jobs.id = $1`
)
