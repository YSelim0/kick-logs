package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type UserRequestRepository struct {
	conn driver.Conn
}

func NewUserRequestRepository(conn driver.Conn) *UserRequestRepository {
	return &UserRequestRepository{conn: conn}
}

func (repo *UserRequestRepository) Create(ctx context.Context, request domain.UserRequest) error {
	batch, err := repo.conn.PrepareBatch(ctx, `INSERT INTO user_requests (
		request_id, type, title, title_lower, message, message_lower,
		channel_slug, channel_slug_lower, channel_display_name, channel_display_name_lower,
		contact, contact_lower, ip_hash, user_agent_hash, created_at
	)`)
	if err != nil {
		return fmt.Errorf("prepare user request insert: %w", err)
	}

	if err := batch.Append(
		request.ID,
		string(request.Type),
		request.Title,
		strings.ToLower(request.Title),
		request.Message,
		strings.ToLower(request.Message),
		nullableString(request.ChannelSlug),
		strings.ToLower(request.ChannelSlug),
		nullableString(request.ChannelDisplayName),
		strings.ToLower(request.ChannelDisplayName),
		nullableString(request.Contact),
		strings.ToLower(request.Contact),
		nullableString(request.IPHash),
		nullableString(request.UserAgentHash),
		request.CreatedAt.UTC(),
	); err != nil {
		return fmt.Errorf("append user request: %w", err)
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("send user request insert: %w", err)
	}
	return nil
}

func (repo *UserRequestRepository) List(
	ctx context.Context,
	filter domain.UserRequestListFilter,
) ([]domain.UserRequestState, error) {
	limit := filter.Limit
	if limit == 0 {
		limit = 50
	}

	where, args := userRequestWhere(filter)
	query := userRequestProjectionSQL(where + `
		ORDER BY created_at DESC, request_id DESC
		LIMIT ? OFFSET ?`)
	args = append(args, limit, filter.Offset)

	rows, err := repo.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list user requests: %w", err)
	}
	defer rows.Close()

	states := make([]domain.UserRequestState, 0, limit)
	for rows.Next() {
		state, err := scanUserRequestState(rows)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user requests: %w", err)
	}
	return states, nil
}

func (repo *UserRequestRepository) Get(ctx context.Context, requestID string) (domain.UserRequestState, error) {
	query := userRequestProjectionSQL("AND request_id = ?")
	state, err := scanUserRequestState(repo.conn.QueryRow(ctx, query, requestID))
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.UserRequestState{}, err
		}
		return domain.UserRequestState{}, fmt.Errorf("get user request: %w", err)
	}
	return state, nil
}

func (repo *UserRequestRepository) ListEvents(ctx context.Context, requestID string) ([]domain.UserRequestEvent, error) {
	rows, err := repo.conn.Query(ctx, `
		SELECT event_id, request_id, event_type, status, note, ifNull(admin_id, 0), created_at
		FROM user_request_events
		WHERE request_id = ?
		ORDER BY created_at ASC, event_id ASC`,
		requestID,
	)
	if err != nil {
		return nil, fmt.Errorf("list user request events: %w", err)
	}
	defer rows.Close()

	events := make([]domain.UserRequestEvent, 0)
	for rows.Next() {
		event, err := scanUserRequestEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user request events: %w", err)
	}
	return events, nil
}

func (repo *UserRequestRepository) AppendEvent(ctx context.Context, event domain.UserRequestEvent) error {
	batch, err := repo.conn.PrepareBatch(ctx, `INSERT INTO user_request_events (
		event_id, request_id, event_type, status, note, admin_id, created_at
	)`)
	if err != nil {
		return fmt.Errorf("prepare user request event insert: %w", err)
	}

	if err := batch.Append(
		event.ID,
		event.RequestID,
		string(event.EventType),
		string(event.Status),
		event.Note,
		nullableInt64(event.AdminID),
		event.CreatedAt.UTC(),
	); err != nil {
		return fmt.Errorf("append user request event: %w", err)
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("send user request event insert: %w", err)
	}
	return nil
}

func userRequestProjectionSQL(suffix string) string {
	return fmt.Sprintf(`
		SELECT
			request_id, type, title, message, channel_slug, channel_display_name,
			contact, ip_hash, user_agent_hash, created_at,
			current_status, is_archived, latest_event_at
		FROM (
			SELECT
				r.request_id AS request_id,
				r.type AS type,
				r.title AS title,
				r.message AS message,
				ifNull(r.channel_slug, '') AS channel_slug,
				ifNull(r.channel_display_name, '') AS channel_display_name,
				ifNull(r.contact, '') AS contact,
				ifNull(r.ip_hash, '') AS ip_hash,
				ifNull(r.user_agent_hash, '') AS user_agent_hash,
				r.created_at AS created_at,
				if(ifNull(e.latest_status, '') = '', 'new', ifNull(e.latest_status, '')) AS current_status,
				toUInt8(ifNull(e.is_archived, 0)) AS is_archived,
				ifNull(e.latest_event_at, r.created_at) AS latest_event_at
			FROM user_requests AS r
			LEFT JOIN (
				SELECT
					request_id,
					argMaxIf(status, created_at, event_type = 'status_changed' AND status != '') AS latest_status,
					countIf(event_type = 'archived') > 0 AS is_archived,
					max(created_at) AS latest_event_at
				FROM user_request_events
				GROUP BY request_id
			) AS e ON e.request_id = r.request_id
		)
		WHERE 1 = 1
		%s`, suffix)
}

func userRequestWhere(filter domain.UserRequestListFilter) (string, []any) {
	where := make([]string, 0, 6)
	args := make([]any, 0, 8)

	if filter.Type != "" {
		where = append(where, "type = ?")
		args = append(args, string(filter.Type))
	}
	if filter.Status != "" {
		where = append(where, "current_status = ?")
		args = append(args, string(filter.Status))
	}
	if filter.Archived != nil {
		archived := uint8(0)
		if *filter.Archived {
			archived = 1
		}
		where = append(where, "is_archived = ?")
		args = append(args, archived)
	}
	if filter.Query != "" {
		query := strings.TrimSpace(filter.Query)
		where = append(where, `(positionCaseInsensitive(title, ?) > 0
			OR positionCaseInsensitive(message, ?) > 0
			OR positionCaseInsensitive(channel_slug, ?) > 0
			OR positionCaseInsensitive(channel_display_name, ?) > 0
			OR positionCaseInsensitive(contact, ?) > 0)`)
		args = append(args, query, query, query, query, query)
	}
	if !filter.Start.IsZero() {
		where = append(where, "created_at >= ?")
		args = append(args, filter.Start.UTC())
	}
	if !filter.End.IsZero() {
		where = append(where, "created_at <= ?")
		args = append(args, filter.End.UTC())
	}

	if len(where) == 0 {
		return "", args
	}
	return "AND " + strings.Join(where, " AND "), args
}

type userRequestStateScanner interface {
	Scan(dest ...any) error
}

func scanUserRequestState(scanner userRequestStateScanner) (domain.UserRequestState, error) {
	var state domain.UserRequestState
	var requestType string
	var currentStatus string
	var isArchived uint8
	if err := scanner.Scan(
		&state.Request.ID,
		&requestType,
		&state.Request.Title,
		&state.Request.Message,
		&state.Request.ChannelSlug,
		&state.Request.ChannelDisplayName,
		&state.Request.Contact,
		&state.Request.IPHash,
		&state.Request.UserAgentHash,
		&state.Request.CreatedAt,
		&currentStatus,
		&isArchived,
		&state.LatestEventAt,
	); err != nil {
		return domain.UserRequestState{}, err
	}
	state.Request.Type = domain.UserRequestType(requestType)
	state.CurrentStatus = domain.UserRequestStatus(currentStatus)
	state.IsArchived = isArchived == 1
	return state, nil
}

type userRequestEventScanner interface {
	Scan(dest ...any) error
}

func scanUserRequestEvent(scanner userRequestEventScanner) (domain.UserRequestEvent, error) {
	var event domain.UserRequestEvent
	var eventType string
	var status string
	if err := scanner.Scan(
		&event.ID,
		&event.RequestID,
		&eventType,
		&status,
		&event.Note,
		&event.AdminID,
		&event.CreatedAt,
	); err != nil {
		return domain.UserRequestEvent{}, fmt.Errorf("scan user request event: %w", err)
	}
	event.EventType = domain.UserRequestEventType(eventType)
	event.Status = domain.UserRequestStatus(status)
	return event, nil
}
