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
    PRIMARY KEY(mailbox_id, flag_id)
) strict;

CREATE TABLE IF NOT EXISTS emails_flags (
    email_id INTEGER NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    flag_id INTEGER NOT NULL REFERENCES flags(id) ON DELETE CASCADE,
    PRIMARY KEY(email_id, flag_id)
) strict;

CREATE TABLE IF NOT EXISTS mailbox_emails (
    mailbox_id INTEGER NOT NULL REFERENCES mailbox(id) ON DELETE CASCADE,
    email_id INTEGER NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    PRIMARY KEY(mailbox_id, email_id)
) strict;

-- ensure that the required mailbox exists

INSERT INTO mailbox (name)
VALUES ("INBOX")
ON CONFLICT DO NOTHING;

INSERT INTO mailbox (name)
VALUES ("Junk")
ON CONFLICT DO NOTHING;

