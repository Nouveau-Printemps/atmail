CREATE TABLE IF NOT EXISTS emails (
	id INTEGER PRIMARY KEY,
	mail TEXT NOT NULL,
	rcpt TEXT NOT NULL,
	subject TEXT NOT NULL,
	spam_score REAL,
	content TEXT NOT NULL,
	thread INTEGER,
	FOREIGN KEY(thread) REFERENCES email(id)
) strict;
