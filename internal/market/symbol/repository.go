package symbol

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/rotisserie/eris"
	"time"
)

type Repository struct {
	conn *pgx.Conn
}

func NewRepository(conn *pgx.Conn) *Repository {
	return &Repository{
		conn: conn,
	}
}

func (r *Repository) FindSymbolProgress(ctx context.Context, s string) (*SymbolProgress, error) {
	sql := `SELECT symbol, last_update FROM progress WHERE symbol = $1`
	rows, errQuery := r.conn.Query(ctx, sql, s)
	if errQuery != nil {
		return nil, eris.Wrap(errQuery, "failed to query progress")
	}

	progress, errCollect := pgx.CollectOneRow(rows, pgx.RowToStructByName[SymbolProgress])
	// if ErrNoRows return nil
	if errors.Is(errCollect, pgx.ErrNoRows) {
		return nil, nil
	}

	if errCollect != nil {
		return nil, errCollect
	}
	return &progress, nil
}

func (r *Repository) UpdateSymbol(ctx context.Context, symbol string, time time.Time) error {
	sql := `INSERT INTO progress (symbol, last_update) VALUES ($1, $2) 
			ON CONFLICT (symbol) DO UPDATE SET last_update = $2`
	_, err := r.conn.Exec(ctx, sql, symbol, time)
	if err != nil {
		return eris.Wrap(err, "failed to update progress")
	}
	return nil
}
