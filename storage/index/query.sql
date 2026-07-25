-- name: ListEmails :many
SELECT * FROM emails
WHERE rcpt_to = ?;

-- name: ListEveryEmails :many
SELECT * FROM emails;

-- name: GetEmail :one
SELECT * FROM emails
WHERE id = ?;

-- name: NewEmail :one
INSERT INTO emails (mail_from, rcpt_to, spam_score, parent, filename, offset)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: DeleteEmail :exec
DELETE FROM emails
WHERE id = ?;

-- name: ListEmailsFrom :many
SELECT * FROM emails
WHERE rcpt_to = ? AND mail_from = ?;

-- name: GetMeta :one
SELECT * FROM meta LIMIT 1;

-- name: SetMeta :exec
INSERT INTO meta (id, last_file, offset)
VALUES (?, ?, ?)
ON CONFLICT DO UPDATE SET
    last_file = excluded.last_file,
    offset = excluded.offset;
