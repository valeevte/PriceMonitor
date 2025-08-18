package products

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) InsertProduct(ctx context.Context, p *Product) (int, error) {
	var id int
	err := r.db.QueryRow(ctx,
		`INSERT INTO products (name, url) VALUES ($1, $2) RETURNING id, created_at`,
		p.Name, p.URL).Scan(&id, &p.CreatedAt)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) GetProducts(ctx context.Context) ([]Product, error) {
	const q = `
SELECT p.id, p.name, p.url, p.created_at,
       (ph.price::double precision) AS price
FROM products p
LEFT JOIN LATERAL (
    SELECT price FROM price_history ph2 WHERE ph2.product_id = p.id ORDER BY recorded_at DESC LIMIT 1
) ph ON true
ORDER BY p.id;
`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []Product
	for rows.Next() {
		var p Product
		var priceNull sql.NullFloat64
		if err := rows.Scan(&p.ID, &p.Name, &p.URL, &p.CreatedAt, &priceNull); err != nil {
			return nil, err
		}
		if priceNull.Valid {
			val := priceNull.Float64
			p.CurrentPrice = &val
		} else {
			p.CurrentPrice = nil
		}
		res = append(res, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

func (r *Repository) GetProductByID(ctx context.Context, id int) (*Product, error) {
	const q = `
SELECT p.id, p.name, p.url, p.created_at,
       (ph.price::double precision) AS price
FROM products p
LEFT JOIN LATERAL (
    SELECT price FROM price_history ph2 WHERE ph2.product_id = p.id ORDER BY recorded_at DESC LIMIT 1
) ph ON true
WHERE p.id = $1
LIMIT 1;
`
	var p Product
	var priceNull sql.NullFloat64
	row := r.db.QueryRow(ctx, q, id)
	if err := row.Scan(&p.ID, &p.Name, &p.URL, &p.CreatedAt, &priceNull); err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	if priceNull.Valid {
		val := priceNull.Float64
		p.CurrentPrice = &val
	} else {
		p.CurrentPrice = nil
	}
	return &p, nil
}

func (r *Repository) InsertPrice(ctx context.Context, productID int, price float64) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO price_history (product_id, price) VALUES ($1, $2)`,
		productID, price)
	return err
}

func (r *Repository) GetPriceHistory(ctx context.Context, productID int) ([]PriceHistory, error) {
	// приводим price к double precision при чтении
	rows, err := r.db.Query(ctx, `
SELECT id, product_id, (price::double precision) AS price, recorded_at
FROM price_history
WHERE product_id = $1
ORDER BY recorded_at DESC
LIMIT 200
`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PriceHistory
	for rows.Next() {
		var ph PriceHistory
		if err := rows.Scan(&ph.ID, &ph.ProductID, &ph.Price, &ph.RecordedAt); err != nil {
			return nil, err
		}
		out = append(out, ph)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) GetAllProductIDs(ctx context.Context) ([]int, error) {
	rows, err := r.db.Query(ctx, `SELECT id FROM products`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *Repository) GetLatestPrice(ctx context.Context, productID int) (float64, bool, error) {
	var price float64
	// приведение для надёжности
	err := r.db.QueryRow(ctx, `SELECT (price::double precision) FROM price_history WHERE product_id = $1 ORDER BY recorded_at DESC LIMIT 1`, productID).Scan(&price)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return price, true, nil
}
