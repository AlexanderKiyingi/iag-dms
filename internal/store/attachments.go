package store

import (
	"context"
	"fmt"

	"github.com/iag/dms/backend/internal/models"
)

func (r *Repository) CreateAttachment(a models.Attachment) (models.Attachment, error) {
	if r.pool != nil {
		return r.pgCreateAttachment(r.bg(), a)
	}
	return r.mem.createAttachment(a)
}

func (r *Repository) GetAttachment(id string) (models.Attachment, error) {
	if r.pool != nil {
		return r.pgGetAttachment(r.bg(), id)
	}
	return r.mem.getAttachment(id)
}

func (r *Repository) ListAttachments(ownerType, ownerID string) []models.Attachment {
	if r.pool != nil {
		return r.pgListAttachments(r.bg(), ownerType, ownerID)
	}
	return r.mem.listAttachments(ownerType, ownerID)
}

func (r *Repository) pgCreateAttachment(ctx context.Context, a models.Attachment) (models.Attachment, error) {
	if a.ID == "" {
		a.ID, _ = r.pgNextID(ctx, "ATT")
	}
	a.CreatedAt = now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO dms_attachments (id, owner_type, owner_id, filename, content_type, size_bytes, storage_key, uploaded_by, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		a.ID, a.OwnerType, a.OwnerID, a.Filename, a.ContentType, a.Size, a.StorageKey, a.UploadedBy, a.CreatedAt)
	return a, err
}

func (r *Repository) pgGetAttachment(ctx context.Context, id string) (models.Attachment, error) {
	var a models.Attachment
	err := r.pool.QueryRow(ctx, `
		SELECT id, owner_type, owner_id, filename, content_type, size_bytes, storage_key, uploaded_by, created_at
		FROM dms_attachments WHERE id=$1`, id).
		Scan(&a.ID, &a.OwnerType, &a.OwnerID, &a.Filename, &a.ContentType, &a.Size, &a.StorageKey, &a.UploadedBy, &a.CreatedAt)
	if err != nil {
		return a, ErrNotFound
	}
	return a, nil
}

func (r *Repository) pgListAttachments(ctx context.Context, ownerType, ownerID string) []models.Attachment {
	rows, err := r.pool.Query(ctx, `
		SELECT id, owner_type, owner_id, filename, content_type, size_bytes, storage_key, uploaded_by, created_at
		FROM dms_attachments
		WHERE ($1='' OR owner_type=$1) AND ($2='' OR owner_id=$2)
		ORDER BY created_at DESC`, ownerType, ownerID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []models.Attachment
	for rows.Next() {
		var a models.Attachment
		if rows.Scan(&a.ID, &a.OwnerType, &a.OwnerID, &a.Filename, &a.ContentType, &a.Size, &a.StorageKey, &a.UploadedBy, &a.CreatedAt) == nil {
			out = append(out, a)
		}
	}
	return out
}

func (m *memoryState) createAttachment(a models.Attachment) (models.Attachment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a.ID == "" {
		a.ID = fmt.Sprintf("ATT-%05d", 1+len(m.attachments))
	}
	a.CreatedAt = now()
	m.attachments = append(m.attachments, a)
	return a, nil
}

func (m *memoryState) getAttachment(id string) (models.Attachment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, a := range m.attachments {
		if a.ID == id {
			return a, nil
		}
	}
	return models.Attachment{}, ErrNotFound
}

func (m *memoryState) listAttachments(ownerType, ownerID string) []models.Attachment {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []models.Attachment
	for _, a := range m.attachments {
		if ownerType != "" && a.OwnerType != ownerType {
			continue
		}
		if ownerID != "" && a.OwnerID != ownerID {
			continue
		}
		out = append(out, a)
	}
	return out
}
