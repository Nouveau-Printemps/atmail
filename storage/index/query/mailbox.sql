-- name: GetMailbox :one
SELECT * FROM mailbox
WHERE name = ?;

-- name: ListMailbox :many
SELECT * FROM mailbox;

-- name: NewMailbox :one
INSERT INTO mailbox (name)
VALUES (?)
RETURNING id;

-- name: RenameMailbox :exec
UPDATE mailbox
SET name = ?
WHERE name = ?;

-- name: DeleteMailbox :exec
DELETE FROM mailbox
WHERE name = ?;

-- name: ListFlags :many
SELECT * FROM flags;

-- name: NewFlag :one
INSERT INTO flags (name)
VALUES (?)
RETURNING id;

-- name: GetMailboxFlags :many
SELECT f.*, m.user_added FROM flags f, mailbox_flags m
WHERE m.mailbox_id = ? and m.flag_id = f.id;

-- name: AddMailboxFlag :exec
INSERT INTO mailbox_flags (mailbox_id, flag_id)
VALUES (?, ?);

-- name: RemoveMailboxFlag :exec
DELETE FROM mailbox_flags
WHERE mailbox_id = ? AND flag_id = ?;

-- name: GetEmailFlags :many
SELECT * FROM flags f
WHERE EXISTS(
    SELECT * FROM emails_flags e WHERE e.email_id = ? and m.flag_id = f.id
);

-- name: AddEmailFlag :exec
INSERT INTO emails_flags (email_id, flag_id)
VALUES (?, ?);

-- name: RemoveEmailFlag :exec
DELETE FROM emails_flags
WHERE email_id = ? AND flag_id = ?;

-- name: AddMailboxEmail :exec
INSERT INTO mailbox_emails (mailbox_id, email_id)
VALUES (?, ?);

-- name: AddInboxEmail :exec
INSERT INTO mailbox_emails (mailbox_id, email_id)
VALUES ((SELECT id FROM mailbox WHERE name = "INBOX"), ?);

-- name: AddJunkEmail :exec
INSERT INTO mailbox_emails (mailbox_id, email_id)
VALUES ((SELECT id FROM mailbox WHERE name = "Junk"), ?);

-- name: RemoveMailboxEmail :exec
DELETE FROM mailbox_emails
WHERE mailbox_id = ? and email_id = ?;

-- name: CountMailboxEmails :one
SELECT COUNT(*) FROM mailbox_emails
WHERE mailbox_id = ?;

-- name: GetLatestMailboxEmails :many
SELECT * FROM emails e
WHERE EXISTS(
    SELECT * FROM mailbox_emails m WHERE m.mailbox_id = ? and m.email_id = e.id
) ORDER BY e.id ASC
LIMIT ? OFFSET ?;
