DELETE FROM mailbox WHERE LENGTH(name) = 0;

ALTER TABLE mailbox ADD CHECK (LENGTH(name) > 0);
