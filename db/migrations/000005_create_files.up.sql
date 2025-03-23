CREATE TABLE IF NOT EXISTS files (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	path TEXT NOT NULL,
	filename TEXT NOT NULL,
	size INTEGER NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (user_id) REFERENCES users (id)
);

CREATE TABLE IF NOT EXISTS file_content (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	file_id INTEGER NOT NULL,
	chunk_index INTEGER NOT NULL,
	content BLOB NOT NULL,
	FOREIGN KEY (file_id) REFERENCES files (id)
);
