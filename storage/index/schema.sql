CREATE TABLE IF NOT EXISTS emails (
	id INTEGER PRIMARY KEY,
	mail_from TEXT NOT NULL,
	rcpt_to TEXT NOT NULL,
	spam_score REAL,
	thread INTEGER,
    file_path TEXT NOT NULL,
    offset INTEGER NOT NULL,
	FOREIGN KEY(thread) REFERENCES email(id)
) strict;
