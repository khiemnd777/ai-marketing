package clients

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/audit"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
)

var (
	ErrInvalid  = errors.New("invalid client")
	ErrNotFound = errors.New("client not found")
	ErrConflict = errors.New("client version conflict")
)

type Client struct {
	ID            uuid.UUID `json:"id"`
	CompanyName   string    `json:"companyName"`
	ContactName   string    `json:"contactName"`
	ContactEmail  *string   `json:"contactEmail"`
	Phone         *string   `json:"phone"`
	Industry      string    `json:"industry"`
	Market        string    `json:"market"`
	InternalNotes string    `json:"internalNotes"`
	Status        string    `json:"status"`
	Version       int64     `json:"version"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type Input struct {
	CompanyName   string  `json:"companyName"`
	ContactName   string  `json:"contactName"`
	ContactEmail  *string `json:"contactEmail"`
	Phone         *string `json:"phone"`
	Industry      string  `json:"industry"`
	Market        string  `json:"market"`
	InternalNotes string  `json:"internalNotes"`
	Version       int64   `json:"version"`
}

type Page struct {
	Items      []Client `json:"items"`
	Number     int      `json:"number"`
	Size       int      `json:"size"`
	TotalItems int64    `json:"totalItems"`
	TotalPages int64    `json:"totalPages"`
}

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

func (s *Service) List(ctx context.Context, page, size int, search, status string) (Page, error) {
	page, size = normalizePage(page, size)
	search = strings.TrimSpace(search)
	status = strings.ToUpper(strings.TrimSpace(status))
	if status != "" && status != "ACTIVE" && status != "ARCHIVED" {
		return Page{}, ErrInvalid
	}
	filterStatus := status == ""
	rows, err := s.pool.Query(ctx, `SELECT id, company_name, contact_name, contact_email, phone, industry, market, internal_notes, status::text, version, created_at, updated_at, count(*) OVER()
		FROM clients WHERE ($1 OR status::text = $2) AND ($3 = '' OR company_name ILIKE '%' || $3 || '%' OR contact_name ILIKE '%' || $3 || '%') ORDER BY company_name, id LIMIT $4 OFFSET $5`, filterStatus, status, search, size, (page-1)*size)
	if err != nil {
		return Page{}, fmt.Errorf("list clients: %w", err)
	}
	defer rows.Close()
	items := make([]Client, 0, size)
	var total int64
	for rows.Next() {
		var item Client
		if err := rows.Scan(&item.ID, &item.CompanyName, &item.ContactName, &item.ContactEmail, &item.Phone, &item.Industry, &item.Market, &item.InternalNotes, &item.Status, &item.Version, &item.CreatedAt, &item.UpdatedAt, &total); err != nil {
			return Page{}, fmt.Errorf("scan client: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return Page{}, err
	}
	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(size) - 1) / int64(size)
	}
	return Page{Items: items, Number: page, Size: size, TotalItems: total, TotalPages: totalPages}, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (Client, error) {
	return scanClient(s.pool.QueryRow(ctx, `SELECT id, company_name, contact_name, contact_email, phone, industry, market, internal_notes, status::text, version, created_at, updated_at FROM clients WHERE id=$1`, id))
}

func (s *Service) Create(ctx context.Context, input Input, actor auth.Principal, metadata auth.ClientMetadata) (Client, error) {
	if err := validate(&input, false); err != nil {
		return Client{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Client{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	item, err := scanClient(tx.QueryRow(ctx, `INSERT INTO clients (company_name,contact_name,contact_email,phone,industry,market,internal_notes,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8) RETURNING id, company_name, contact_name, contact_email, phone, industry, market, internal_notes, status::text, version, created_at, updated_at`, input.CompanyName, input.ContactName, input.ContactEmail, input.Phone, input.Industry, input.Market, input.InternalNotes, actor.UserID))
	if err != nil {
		return Client{}, fmt.Errorf("create client: %w", err)
	}
	if err := recordAudit(ctx, tx, actor, metadata, "client.created", item.ID, nil, item); err != nil {
		return Client{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Client{}, err
	}
	return item, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, input Input, actor auth.Principal, metadata auth.ClientMetadata) (Client, error) {
	if err := validate(&input, true); err != nil {
		return Client{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Client{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := scanClient(tx.QueryRow(ctx, `SELECT id, company_name, contact_name, contact_email, phone, industry, market, internal_notes, status::text, version, created_at, updated_at FROM clients WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return Client{}, err
	}
	item, err := scanClient(tx.QueryRow(ctx, `UPDATE clients SET company_name=$2,contact_name=$3,contact_email=$4,phone=$5,industry=$6,market=$7,internal_notes=$8,version=version+1,updated_by=$9,updated_at=now() WHERE id=$1 AND version=$10 RETURNING id, company_name, contact_name, contact_email, phone, industry, market, internal_notes, status::text, version, created_at, updated_at`, id, input.CompanyName, input.ContactName, input.ContactEmail, input.Phone, input.Industry, input.Market, input.InternalNotes, actor.UserID, input.Version))
	if errors.Is(err, pgx.ErrNoRows) {
		return Client{}, ErrConflict
	}
	if err != nil {
		return Client{}, err
	}
	if err := recordAudit(ctx, tx, actor, metadata, "client.updated", id, before, item); err != nil {
		return Client{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Client{}, err
	}
	return item, nil
}

func (s *Service) SetStatus(ctx context.Context, id uuid.UUID, status string, version int64, actor auth.Principal, metadata auth.ClientMetadata) (Client, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	if (status != "ACTIVE" && status != "ARCHIVED") || version < 1 {
		return Client{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Client{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := scanClient(tx.QueryRow(ctx, `SELECT id, company_name, contact_name, contact_email, phone, industry, market, internal_notes, status::text, version, created_at, updated_at FROM clients WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return Client{}, err
	}
	item, err := scanClient(tx.QueryRow(ctx, `UPDATE clients SET status=$2::lifecycle_status,archived_at=CASE WHEN $2='ARCHIVED' THEN now() ELSE NULL END,version=version+1,updated_by=$3,updated_at=now() WHERE id=$1 AND version=$4 RETURNING id, company_name, contact_name, contact_email, phone, industry, market, internal_notes, status::text, version, created_at, updated_at`, id, status, actor.UserID, version))
	if errors.Is(err, pgx.ErrNoRows) {
		return Client{}, ErrConflict
	}
	if err != nil {
		return Client{}, err
	}
	action := "client.reactivated"
	if status == "ARCHIVED" {
		action = "client.archived"
	}
	if err := recordAudit(ctx, tx, actor, metadata, action, id, before, item); err != nil {
		return Client{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Client{}, err
	}
	return item, nil
}

type rowScanner interface{ Scan(...any) error }

func scanClient(row rowScanner) (Client, error) {
	var item Client
	err := row.Scan(&item.ID, &item.CompanyName, &item.ContactName, &item.ContactEmail, &item.Phone, &item.Industry, &item.Market, &item.InternalNotes, &item.Status, &item.Version, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Client{}, ErrNotFound
	}
	return item, err
}
func validate(input *Input, version bool) error {
	input.CompanyName = strings.TrimSpace(input.CompanyName)
	input.ContactName = strings.TrimSpace(input.ContactName)
	input.Industry = strings.TrimSpace(input.Industry)
	input.Market = strings.TrimSpace(input.Market)
	input.InternalNotes = strings.TrimSpace(input.InternalNotes)
	if len(input.CompanyName) < 2 || len(input.CompanyName) > 200 || len(input.ContactName) > 160 || len(input.InternalNotes) > 10000 || (version && input.Version < 1) {
		return ErrInvalid
	}
	if input.ContactEmail != nil {
		v := strings.ToLower(strings.TrimSpace(*input.ContactEmail))
		if v == "" {
			input.ContactEmail = nil
		} else {
			if _, err := mail.ParseAddress(v); err != nil || len(v) > 320 {
				return ErrInvalid
			}
			input.ContactEmail = &v
		}
	}
	if input.Phone != nil {
		v := strings.TrimSpace(*input.Phone)
		if len(v) > 40 {
			return ErrInvalid
		}
		if v == "" {
			input.Phone = nil
		} else {
			input.Phone = &v
		}
	}
	return nil
}
func normalizePage(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 25
	}
	if size > 100 {
		size = 100
	}
	return page, size
}
func recordAudit(ctx context.Context, tx pgx.Tx, actor auth.Principal, metadata auth.ClientMetadata, action string, id uuid.UUID, before, after any) error {
	return audit.Record(ctx, db.New(tx), audit.Event{ActorID: uuid.NullUUID{UUID: actor.UserID, Valid: true}, Action: action, EntityType: "client", EntityID: uuid.NullUUID{UUID: id, Valid: true}, RequestID: metadata.RequestID, IPAddress: metadata.IPAddress, UserAgent: metadata.UserAgent, Outcome: "SUCCESS", Before: before, After: after})
}
