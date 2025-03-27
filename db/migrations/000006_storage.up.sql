CREATE TABLE IF NOT EXISTS buckets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS objects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bucket_id INTEGER NOT NULL,
    key TEXT NOT NULL,
    data BLOB NOT NULL,
    content_type TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    etag TEXT NOT NULL,
    FOREIGN KEY (bucket_id) REFERENCES buckets(id),
    UNIQUE(bucket_id, key)
);

CREATE TABLE IF NOT EXISTS multipart_uploads (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bucket_id INTEGER NOT NULL,
    key TEXT NOT NULL,
    upload_id TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (bucket_id) REFERENCES buckets(id)
);

CREATE TABLE IF NOT EXISTS multipart_parts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    upload_id TEXT NOT NULL,
    part_number INTEGER NOT NULL,
    data BLOB NOT NULL,
    etag TEXT NOT NULL,
    FOREIGN KEY (upload_id) REFERENCES multipart_uploads(upload_id),
    UNIQUE(upload_id, part_number)
);
