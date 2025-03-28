CREATE TABLE IF NOT EXISTS chats (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	title TEXT NOT NULL,
	timestamp INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS chats_pins (
	id INTEGER PRIMARY KEY,
	chat_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE
	UNIQUE(chat_id, user_id)
);

CREATE TABLE messages (
	id TEXT PRIMARY KEY,
	chat_id TEXT NOT NULL,
	sender TEXT NOT NULL, -- user:<id> | model:<id>
	sender_role TEXT NOT NULL, -- 'user' | 'assistant' | 'tool' | 'system'
	content TEXT NOT NULL,
	timestamp INTEGER NOT NULL,
	tools JSON,
	FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE
);

CREATE INDEX idx_chats_user_id ON chats(user_id);
CREATE INDEX idx_chats_pinned_user_id ON chats_pins(user_id);

CREATE INDEX idx_messages_chat_id ON messages(chat_id);
CREATE INDEX idx_messages_timestamp ON messages(timestamp);
