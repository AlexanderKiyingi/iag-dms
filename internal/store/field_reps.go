package store

import (
	"context"
	"strings"

	"github.com/iag/dms/backend/internal/models"
)

func normaliseFieldRep(in *models.FieldRep) error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return invalid("a field rep needs a name")
	}
	in.BeatID = strings.TrimSpace(in.BeatID)
	in.Region = strings.TrimSpace(in.Region)
	in.Level = strings.TrimSpace(in.Level)
	in.Status = lower(in.Status)
	if in.Status == "" {
		in.Status = "active"
	}
	return nil
}

func (r *Repository) GetRep(id string) (models.FieldRep, error) {
	if r.pool != nil {
		return r.pgGetRep(r.bg(), id)
	}
	r.mem.mu.RLock()
	defer r.mem.mu.RUnlock()
	for _, rep := range r.mem.reps {
		if rep.ID == id {
			return rep, nil
		}
	}
	return models.FieldRep{}, ErrNotFound
}

func (r *Repository) CreateRep(in models.FieldRep) (models.FieldRep, error) {
	if err := normaliseFieldRep(&in); err != nil {
		return models.FieldRep{}, err
	}
	if r.pool != nil {
		return r.pgCreateRep(r.bg(), in)
	}
	r.mem.mu.Lock()
	defer r.mem.mu.Unlock()
	if in.ID == "" {
		seq := len(r.mem.reps) + 1
		in.ID = r.mem.nextID("FF", &seq)
	}
	for _, existing := range r.mem.reps {
		if existing.ID == in.ID {
			return models.FieldRep{}, conflictf("field rep %s already exists", in.ID)
		}
	}
	r.mem.reps = append(r.mem.reps, in)
	return in, nil
}

func (r *Repository) UpdateRep(id string, in models.FieldRep) (models.FieldRep, error) {
	if err := normaliseFieldRep(&in); err != nil {
		return models.FieldRep{}, err
	}
	if r.pool != nil {
		return r.pgUpdateRep(r.bg(), id, in)
	}
	r.mem.mu.Lock()
	defer r.mem.mu.Unlock()
	for i, existing := range r.mem.reps {
		if existing.ID == id {
			in.ID = id
			r.mem.reps[i] = in
			return in, nil
		}
	}
	return models.FieldRep{}, ErrNotFound
}

func (r *Repository) DeleteRep(id string) error {
	if r.pool != nil {
		_, err := r.pool.Exec(r.bg(), `DELETE FROM dms_field_reps WHERE id=$1`, id)
		if err != nil {
			return err
		}
		return nil
	}
	r.mem.mu.Lock()
	defer r.mem.mu.Unlock()
	return deleteByID(&r.mem.reps, id, func(rep models.FieldRep) string { return rep.ID })
}

func (r *Repository) pgGetRep(ctx context.Context, id string) (models.FieldRep, error) {
	var rep models.FieldRep
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, beat_id, region, level, status FROM dms_field_reps WHERE id=$1`, id).
		Scan(&rep.ID, &rep.Name, &rep.BeatID, &rep.Region, &rep.Level, &rep.Status)
	if err != nil {
		return rep, ErrNotFound
	}
	return rep, nil
}

func (r *Repository) pgCreateRep(ctx context.Context, in models.FieldRep) (models.FieldRep, error) {
	if in.ID == "" {
		id, err := r.pgNextID(ctx, "FF")
		if err != nil {
			return models.FieldRep{}, err
		}
		in.ID = id
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO dms_field_reps (id, name, beat_id, region, level, status)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		in.ID, in.Name, in.BeatID, in.Region, in.Level, in.Status)
	return in, err
}

func (r *Repository) pgUpdateRep(ctx context.Context, id string, in models.FieldRep) (models.FieldRep, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE dms_field_reps SET name=$2, beat_id=$3, region=$4, level=$5, status=$6 WHERE id=$1`,
		id, in.Name, in.BeatID, in.Region, in.Level, in.Status)
	if err != nil {
		return models.FieldRep{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.FieldRep{}, ErrNotFound
	}
	in.ID = id
	return in, nil
}
