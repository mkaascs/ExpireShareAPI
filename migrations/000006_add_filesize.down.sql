-- Delete column file_size into files table
-- All data will be deleted nonreturnable. Make back up
ALTER TABLE files DROP COLUMN file_size;