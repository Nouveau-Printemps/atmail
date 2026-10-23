CREATE TABLE IF NOT EXISTS mailbox (
	id INTEGER PRIMARY KEY,
	name TEXT NOT NULL UNIQUE
) strict;

CREATE TABLE IF NOT EXISTS flags (
	id INTEGER PRIMARY KEY,
	name TEXT NOT NULL UNIQUE
) strict;

CREATE TABLE IF NOT EXISTS mailbox_flags (
	mailbox_id INTEGER NOT NULL REFERENCES mailbox(id) ON DELETE CASCADE,
	flag_id INTEGER NOT NULL REFERENCES flags(id) ON DELETE CASCADE,
	user_added INTEGER NOT NULL DEFAULT false CHECK (user_added IN (0, 1)),
	PRIMARY KEY(mailbox_id, flag_id)
) strict;

CREATE TABLE IF NOT EXISTS emails_flags (
	email_id INTEGER NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
	flag_id INTEGER NOT NULL REFERENCES flags(id) ON DELETE CASCADE,
	user_added INTEGER  NOT NULL DEFAULT false CHECK (user_added IN (0, 1)),
	PRIMARY KEY(email_id, flag_id)
) strict;

CREATE TABLE IF NOT EXISTS mailbox_emails (
	mailbox_id INTEGER NOT NULL REFERENCES mailbox(id) ON DELETE CASCADE,
	email_id INTEGER NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
	PRIMARY KEY(mailbox_id, email_id)
) strict;
