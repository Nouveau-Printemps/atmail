-- name: GetOrCreateMailbox :one
INSERT INTO mailbox (name)
VALUES (?)
ON CONFLICT DO UPDATE SET name = name
RETURNING *;

-- name: ListMailbox :many
SELECT m.*, (
	SELECT COUNT(*) FROM mailbox_emails e WHERE e.mailbox_id = m.id
) FROM mailbox m;

-- name: NewMailbox :one
INSERT INTO mailbox (name)
VALUES (?)
RETURNING id;

-- name: RenameMailbox :exec
UPDATE mailbox
SET name = ?
WHERE id = ?;

-- name: DeleteMailbox :exec
DELETE FROM mailbox
WHERE id = ?;

-- name: ListFlags :many
SELECT * FROM flags;

-- name: NewFlag :one
INSERT INTO flags (name)
VALUES (?)
RETURNING id;

-- name: ListMailboxFlags :many
SELECT f.*, m.user_added FROM flags f, mailbox_flags m
WHERE m.mailbox_id = ? and m.flag_id = f.id;

-- name: AddMailboxFlag :exec
INSERT INTO mailbox_flags (mailbox_id, flag_id)
VALUES (?, ?);

-- name: RemoveMailboxFlag :exec
DELETE FROM mailbox_flags
WHERE mailbox_id = ? AND flag_id = ?;

-- name: ListEmailFlags :many
SELECT * FROM flags f
WHERE EXISTS(
	SELECT * FROM emails_flags e WHERE e.email_id = ? and e.flag_id = f.id
);

-- name: AddEmailFlag :exec
INSERT INTO emails_flags (email_id, flag_id)
VALUES (?, ?)
ON CONFLICT DO NOTHING;

-- name: RemoveEmailFlag :exec
DELETE FROM emails_flags
WHERE email_id = ? AND flag_id = ?;

-- name: AddMailboxEmail :exec
INSERT INTO mailbox_emails (mailbox_id, email_id)
VALUES (?, ?);

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

-- name: GetMailboxEmails :many
SELECT * FROM emails e
WHERE EXISTS(
	SELECT * FROM mailbox_emails m WHERE m.mailbox_id = ? and m.email_id = e.id
) AND e.id IN (sqlc.slice('ids'));

-- name: ListMailboxEmails :many
SELECT * FROM emails e
WHERE EXISTS(
	SELECT * FROM mailbox_emails m WHERE m.mailbox_id = ? and m.email_id = e.id
);

-- name: RemoveMailboxEmails :exec
DELETE FROM emails
WHERE EXISTS(
	SELECT * FROM mailbox_emails m WHERE m.mailbox_id = ? and m.email_id = id
) AND id IN (sqlc.slice('ids'));

-- name: ListEmailsNoMailbox :many
SELECT * FROM emails e
WHERE NOT EXISTS(
	SELECT * FROM mailbox_emails m WHERE m.email_id = e.id
);

-- name: ListEmailsWithFlag :many
SELECT * FROM emails e
WHERE EXISTS (
	SELECT * FROM emails_flags f WHERE f.flag_id = ? and f.email_id = e.id
);

-- name: RemoveEmailsWithFlag :exec
DELETE FROM emails
WHERE EXISTS (
	SELECT * FROM emails_flags f WHERE f.email_id = id AND f.flag_id = ?
);

-- name: RemoveEmailFlags :exec
DELETE FROM emails_flags
WHERE email_id = ?;

-- name: AddEmailFlagName :exec
INSERT INTO emails_flags (email_id, flag_id)
VALUES (?, (
		SELECT id FROM flags WHERE name = ?
))
ON CONFLICT DO NOTHING;

-- name: RemoveEmailFlagName :exec
DELETE FROM emails_flags
WHERE email_id = ? AND flag_id = (
	SELECT id FROM flags WHERE name = ?
);

-- name: CountEmailsWithFlagInMailbox :one
SELECT COUNT(*) FROM emails_flags e
WHERE e.flag_id = ? AND EXISTS(
	SELECT * FROM mailbox_emails m WHERE m.email_id = e.email_id AND m.mailbox_id = ?
);

-- name: GetSequence :one
SELECT COUNT(*) FROM mailbox_emails
WHERE mailbox_id = ? AND email_id <= ?;

-- name: FromSequence :one
SELECT email_id FROM mailbox_emails
WHERE mailbox_id = ?
ORDER BY email_id DESC
LIMIT 1 OFFSET ?-1;

-- name: HasMailboxChildren :one
SELECT COUNT(*) > 1 FROM mailbox
WHERE name LIKE ?;
