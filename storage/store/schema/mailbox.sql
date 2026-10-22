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

-- ensure that the required mailboxes exist
BEGIN IMMEDIATE TRANSACTION;
	INSERT INTO mailbox (id, name)
	VALUES (1, 'INBOX')
	ON CONFLICT DO NOTHING;

	INSERT INTO mailbox (id, name)
	VALUES (2, 'Junk')
	ON CONFLICT DO NOTHING;

	INSERT INTO mailbox (id, name)
	VALUES (3, 'Sent')
	ON CONFLICT DO NOTHING;

	INSERT INTO mailbox (id, name)
	VALUES (4, 'Trash')
	ON CONFLICT DO NOTHING;
END;

-- ensure that the required flags exist
BEGIN IMMEDIATE TRANSACTION;
	INSERT INTO flags (id, name)
	VALUES (1, '\Seen')
	ON CONFLICT DO NOTHING;

	INSERT INTO flags (id, name)
	VALUES (2, '\Answered')
	ON CONFLICT DO NOTHING;

	INSERT INTO flags (id, name)
	VALUES (3, '\Flagged')
	ON CONFLICT DO NOTHING;

	INSERT INTO flags (id, name)
	VALUES (4, '\Deleted')
	ON CONFLICT DO NOTHING;

	INSERT INTO flags (id, name)
	VALUES (5, '\Draft')
	ON CONFLICT DO NOTHING;

	INSERT INTO flags (id, name)
	VALUES (6, '$Forwarded')
	ON CONFLICT DO NOTHING;

	INSERT INTO flags (id, name)
	VALUES (7, '$MDNSent')
	ON CONFLICT DO NOTHING;

	INSERT INTO flags (id, name)
	VALUES (8, '$Junk')
	ON CONFLICT DO NOTHING;

	INSERT INTO flags (id, name)
	VALUES (9, '$NotJunk')
	ON CONFLICT DO NOTHING;

	INSERT INTO flags (id, name)
	VALUES (10, '$Phishing')
	ON CONFLICT DO NOTHING;
END;

-- set required flags for mailboxes
BEGIN IMMEDIATE TRANSACTION;
	INSERT INTO mailbox_flags (mailbox_id, flag_id)
	VALUES (1, 1)
	ON CONFLICT DO NOTHING;

	INSERT INTO mailbox_flags (mailbox_id, flag_id)
	VALUES (1, 2)
	ON CONFLICT DO NOTHING;

	INSERT INTO mailbox_flags (mailbox_id, flag_id)
	VALUES (1, 3)
	ON CONFLICT DO NOTHING;

	INSERT INTO mailbox_flags (mailbox_id, flag_id)
	VALUES (2, 8)
	ON CONFLICT DO NOTHING;

	INSERT INTO mailbox_flags (mailbox_id, flag_id)
	VALUES (4, 4)
	ON CONFLICT DO NOTHING;
END;
