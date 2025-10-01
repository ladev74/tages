package postgres

const (
	querySaveFileInfo = `INSERT INTO schema_files.table_files (id, name, created_at, updated_at, status) VALUES ($1, $2, $3, $4, $5)`

	querySetSuccessStatus = `UPDATE schema_files.table_files SET status = $1 WHERE id = $2`

	queryDeleteFileInfo = `DELETE FROM schema_files.table_files WHERE id = $1`

	queryListFilesInfo = `SELECT name, created_at, updated_at 
						FROM schema_files.table_files ORDER BY created_at DESC LIMIT $1	OFFSET $2`
)
