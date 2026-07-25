-- name: ListEmails :many
SELECT * FROM emails
WHERE rcpt = ?;

-- name: GetEmail :one
SELECT * FROM emails
WHERE id = ?;

-- name: SetEmail :one
INSERT INTO emails (mail, rcpt, subject, spam_score, content, thread)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: DeleteEmail :exec
DELETE FROM emails
WHERE id = ?;

-- name: ListEmailsFrom :many
SELECT * FROM emails
WHERE rcpt = ? AND mail = ?;
