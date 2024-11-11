package db

const (
	initTableQuery = `CREATE TABLE IF NOT EXISTS jobs (
		id text PRIMARY KEY,
		state text NOT NULL,
		source_url text NOT NULL,
		timestamp bigint NOT NULL
	)`
)

const (
	createJobQuery = `INSERT INTO jobs (id, state, source_url, timestamp) VALUES ($1, $2, $3, $4)`
	readJobQuery   = `SELECT * FROM jobs WHERE jobs.id = $1 ORDER BY timestamp DESC LIMIT 1`
)
