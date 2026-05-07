package mysql

import (
	"context"
	"database/sql"
	"errors"
	"expire-share/internal/domain/dto/files/commands"
	"expire-share/internal/domain/entities"
	domainErrors "expire-share/internal/domain/entities/errors"
	"expire-share/internal/domain/interfaces/tx"
	"expire-share/internal/lib/log/sl"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
)

const (
	duplicateEntryErrCode = 1062
)

type FileRepo struct {
	DB  *sql.DB
	log *slog.Logger
}

func NewFileRepo(db *sql.DB, log *slog.Logger) *FileRepo {
	return &FileRepo{DB: db, log: log}
}

func (fr *FileRepo) BeginTx(ctx context.Context) (tx.Tx, error) {
	return fr.DB.BeginTx(ctx, nil)
}

func (fr *FileRepo) AddFileTx(ctx context.Context, tx tx.Tx, command commands.AddFile) (int64, error) {
	const fn = "repository.mysql.FileRepo.AddFile"

	sqlTx, ok := tx.(*sql.Tx)
	if !ok {
		return 0, fmt.Errorf("%s: failed to convert tx to sql", fn)
	}

	currentTime := time.Now()
	res, err := sqlTx.ExecContext(ctx, `INSERT INTO files(file_name, file_size, alias, downloads_left, loaded_at, expires_at, password_hash, user_id) VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
		command.Filename,
		command.Filesize,
		command.Alias,
		command.MaxDownloads,
		currentTime,
		currentTime.Add(command.TTL),
		command.PasswordHash,
		command.UserID)

	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == duplicateEntryErrCode {
			return 0, domainErrors.ErrAliasTaken
		}

		return 0, fmt.Errorf("%s: failed to exec sql: %w", fn, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get last insert id: %w", fn, err)
	}

	return id, nil
}

func (fr *FileRepo) GetFileByAlias(ctx context.Context, alias string) (*entities.File, error) {
	const fn = "repository.mysql.FileRepo.GetFileByAlias"

	var file entities.File
	err := fr.DB.QueryRowContext(ctx, `SELECT file_name, file_size, alias, downloads_left, loaded_at, expires_at, password_hash, user_id FROM files WHERE alias = ? AND expires_at > NOW()`, alias).Scan(
		&file.Name,
		&file.Size,
		&file.Alias,
		&file.DownloadsLeft,
		&file.LoadedAt,
		&file.ExpiresAt,
		&file.PasswordHash,
		&file.UserID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainErrors.ErrFileNotFound
		}

		return nil, fmt.Errorf("%s: failed to query sql: %w", fn, err)
	}

	return &file, nil
}

func (fr *FileRepo) GetFilesByUserID(ctx context.Context, command commands.GetAllFiles) ([]entities.File, int, error) {
	const fn = "repository.mysql.FileRepo.GetFilesByUserID"
	log := fr.log.With(slog.String("fn", fn))

	var total int
	err := fr.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM files WHERE user_id = ? AND expires_at > NOW()`, command.UserID).
		Scan(&total)

	if err != nil {
		return nil, 0, fmt.Errorf("%s: failed to query sql: %w", fn, err)
	}

	if total == 0 {
		return []entities.File{}, 0, nil
	}

	offset := (command.Page - 1) * command.Limit
	rows, err := fr.DB.QueryContext(ctx, `SELECT file_name, file_size, alias, downloads_left, loaded_at, expires_at, password_hash, user_id FROM files WHERE user_id = ? AND expires_at > NOW() LIMIT ? OFFSET ?`,
		command.UserID, command.Limit, offset)

	if err != nil {
		return nil, 0, fmt.Errorf("%s: failed to query sql: %w", fn, err)
	}

	defer func(rows *sql.Rows) {
		if err := rows.Close(); err != nil {
			log.Warn("failed to close rows", sl.Error(err))
		}
	}(rows)

	var files []entities.File

	for rows.Next() {
		var file entities.File
		if err := rows.Scan(&file.Name,
			&file.Size,
			&file.Alias,
			&file.DownloadsLeft,
			&file.LoadedAt,
			&file.ExpiresAt,
			&file.PasswordHash,
			&file.UserID); err != nil {
			return files, total, fmt.Errorf("%s: failed to scan row: %w", fn, err)
		}

		files = append(files, file)
	}

	if err := rows.Err(); err != nil {
		return files, total, fmt.Errorf("%s: failed to iterate rows: %w", fn, err)
	}

	return files, total, nil
}

func (fr *FileRepo) GetFilesStatByUserID(ctx context.Context, userId int64) (*entities.FilesStat, error) {
	const fn = "repository.mysql.FileRepo.GetFilesStatByUserID"

	stat := entities.FilesStat{UserID: userId}
	err := fr.DB.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(file_size), 0) FROM files WHERE user_id = ?`, userId).
		Scan(&stat.Count, &stat.Size)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainErrors.ErrUserNotFound
		}

		return nil, fmt.Errorf("%s: failed to query sql: %w", fn, err)
	}

	return &stat, nil
}

func (fr *FileRepo) DecrementDownloadsByAliasTx(ctx context.Context, tx tx.Tx, alias string) (int16, error) {
	const fn = "repository.mysql.FileRepo.DecrementDownloadsByAlias"

	sqlTx, ok := tx.(*sql.Tx)
	if !ok {
		return 0, fmt.Errorf("%s: failed to convert tx to sql", fn)
	}

	var downloadsLeft int16
	err := sqlTx.QueryRowContext(ctx, `SELECT downloads_left FROM files WHERE alias = ? AND expires_at > NOW()`, alias).
		Scan(&downloadsLeft)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, domainErrors.ErrFileNotFound
		}

		return 0, fmt.Errorf("%s: failed to exec sql: %w", fn, err)
	}

	if downloadsLeft == 0 {
		return 0, domainErrors.ErrNoDownloadsLeft
	}

	downloadsLeft--
	_, err = sqlTx.ExecContext(ctx, `UPDATE files SET downloads_left = ? WHERE alias = ?`, downloadsLeft, alias)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to exec sql: %w", fn, err)
	}

	return downloadsLeft, nil
}

func (fr *FileRepo) DeleteFileTx(ctx context.Context, tx tx.Tx, alias string) error {
	const fn = "repository.mysql.FileRepo.DeleteFile"

	sqlTx, ok := tx.(*sql.Tx)
	if !ok {
		return fmt.Errorf("%s: failed to convert tx to sql", fn)
	}

	res, err := sqlTx.ExecContext(ctx, `DELETE FROM files WHERE alias = ? AND expires_at > NOW()`, alias)
	if err != nil {
		return fmt.Errorf("%s: failed to exec sql: %w", fn, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: failed to affect rows: %w", fn, err)
	}

	if rowsAffected == 0 {
		return domainErrors.ErrFileNotFound
	}

	return nil
}

func (fr *FileRepo) DeleteExpiredFilesTx(ctx context.Context, tx tx.Tx, limit int) ([]string, error) {
	const fn = "repository.mysql.FileRepo.DeleteExpiredFiles"
	log := fr.log.With(slog.String("fn", fn))

	sqlTx, ok := tx.(*sql.Tx)
	if !ok {
		return nil, fmt.Errorf("%s: failed to convert tx to sql", fn)
	}

	rows, err := sqlTx.QueryContext(ctx, `SELECT alias FROM files WHERE expires_at < NOW() LIMIT ? FOR UPDATE`, limit)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to exec sql: %w", fn, err)
	}

	defer func(rows *sql.Rows) {
		if err := rows.Close(); err != nil {
			log.Warn("failed to close rows", sl.Error(err))
		}
	}(rows)

	var aliases []string
	for rows.Next() {
		var alias string
		if err := rows.Scan(&alias); err != nil {
			return nil, fmt.Errorf("%s: failed to scan alias: %w", fn, err)
		}

		aliases = append(aliases, alias)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", fn, err)
	}

	if len(aliases) == 0 {
		return aliases, nil
	}

	stmt, err := sqlTx.PrepareContext(ctx, `DELETE FROM files WHERE alias = ?`)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to prepare stmt: %w", fn, err)
	}

	defer func(stmt *sql.Stmt) {
		if err := stmt.Close(); err != nil {
			log.Warn("failed to close stmt", sl.Error(err))
		}
	}(stmt)

	for _, alias := range aliases {
		if _, err := stmt.Exec(alias); err != nil {
			return nil, fmt.Errorf("%s: failed to exec sql: %w", fn, err)
		}
	}

	return aliases, nil
}
