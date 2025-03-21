CREATE TABLE IF NOT EXISTS reminders (
    id TEXT NOT NULL PRIMARY KEY,
    message TEXT,
    description TEXT,
    due_time DATETIME,
    completed BOOLEAN DEFAULT FALSE,
	webhook_url TEXT
);
