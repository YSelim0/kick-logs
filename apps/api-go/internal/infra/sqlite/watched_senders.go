package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type WatchedSenderRepository struct {
	db *sql.DB
}

func NewWatchedSenderRepository(db *sql.DB) *WatchedSenderRepository {
	return &WatchedSenderRepository{db: db}
}

func (repo *WatchedSenderRepository) Create(ctx context.Context, username string) (domain.WatchedSender, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return domain.WatchedSender{}, fmt.Errorf("username is required")
	}

	now := formatTime(time.Now().UTC())
	result, err := repo.db.ExecContext(
		ctx,
		`INSERT INTO watched_senders (username, created_at) VALUES (?, ?)`,
		username, now,
	)
	if err != nil {
		return domain.WatchedSender{}, fmt.Errorf("insert watched sender: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.WatchedSender{}, fmt.Errorf("read watched sender id: %w", err)
	}
	return repo.GetByID(ctx, id)
}

func (repo *WatchedSenderRepository) GetByID(ctx context.Context, id int64) (domain.WatchedSender, error) {
	row := repo.db.QueryRowContext(
		ctx,
		`SELECT id, username, created_at FROM watched_senders WHERE id = ?`,
		id,
	)
	return scanWatchedSender(row)
}

func (repo *WatchedSenderRepository) Delete(ctx context.Context, id int64) error {
	result, err := repo.db.ExecContext(ctx, `DELETE FROM watched_senders WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete watched sender: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read delete result: %w", err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (repo *WatchedSenderRepository) List(ctx context.Context) ([]domain.WatchedSender, error) {
	rows, err := repo.db.QueryContext(
		ctx,
		`SELECT id, username, created_at FROM watched_senders ORDER BY username COLLATE NOCASE`,
	)
	if err != nil {
		return nil, fmt.Errorf("list watched senders: %w", err)
	}
	defer rows.Close()

	senders := make([]domain.WatchedSender, 0)
	for rows.Next() {
		sender, err := scanWatchedSenderRows(rows)
		if err != nil {
			return nil, err
		}
		senders = append(senders, sender)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate watched senders: %w", err)
	}
	return senders, nil
}

func (repo *WatchedSenderRepository) ListUsernames(ctx context.Context) ([]string, error) {
	senders, err := repo.List(ctx)
	if err != nil {
		return nil, err
	}
	usernames := make([]string, 0, len(senders))
	for _, sender := range senders {
		usernames = append(usernames, sender.Username)
	}
	return usernames, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanWatchedSender(row *sql.Row) (domain.WatchedSender, error) {
	return scanWatchedSenderInto(row)
}

func scanWatchedSenderRows(rows *sql.Rows) (domain.WatchedSender, error) {
	return scanWatchedSenderInto(rows)
}

func scanWatchedSenderInto(scanner rowScanner) (domain.WatchedSender, error) {
	var (
		sender    domain.WatchedSender
		createdAt string
	)
	if err := scanner.Scan(&sender.ID, &sender.Username, &createdAt); err != nil {
		return domain.WatchedSender{}, err
	}
	sender.CreatedAt = parseTime(createdAt)
	return sender, nil
}
