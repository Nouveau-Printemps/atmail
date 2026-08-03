CREATE TABLE IF NOT EXISTS emails (
	id INTEGER PRIMARY KEY,
	mail_from TEXT NOT NULL,
	rcpt_to TEXT NOT NULL,
	spam_score REAL,
	parent INTEGER,
    filename TEXT NOT NULL,
    offset INTEGER NOT NULL,
    internal_date INTEGER NOT NULL,
	FOREIGN KEY(parent) REFERENCES emails(id) ON DELETE SET NULL
) strict;

CREATE TABLE IF NOT EXISTS meta (
    id INTEGER PRIMARY KEY,
    last_file TEXT,
    offset INTEGER
) strict;
