-- name: ListEmails :many
SELECT * FROM emails
WHERE rcpt_to = ?;

-- name: ListEveryEmails :many
SELECT * FROM emails;

-- name: GetEmail :one
SELECT * FROM emails
WHERE id = ?;

-- name: NewEmail :one
INSERT INTO emails (mail_from, rcpt_to, spam_score, thread, filename, offset)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: DeleteEmail :exec
DELETE FROM emails
WHERE id = ?;

-- name: ListEmailsFrom :many
SELECT * FROM emails
WHERE rcpt_to = ? AND mail_from = ?;
