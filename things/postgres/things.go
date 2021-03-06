package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // required for DB access
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/things"
)

var _ things.ThingRepository = (*thingRepository)(nil)

type thingRepository struct {
	db  *sql.DB
	log logger.Logger
}

// NewThingRepository instantiates a PostgreSQL implementation of thing
// repository.
func NewThingRepository(db *sql.DB, log logger.Logger) things.ThingRepository {
	return &thingRepository{db: db, log: log}
}

func (tr thingRepository) Save(thing things.Thing) (uint64, error) {
	q := `INSERT INTO things (owner, type, name, key, metadata) VALUES ($1, $2, $3, $4, $5) RETURNING id`

	if err := tr.db.QueryRow(q, thing.Owner, thing.Type, thing.Name, thing.Key, thing.Metadata).Scan(&thing.ID); err != nil {
		return 0, err
	}

	return thing.ID, nil
}

func (tr thingRepository) Update(thing things.Thing) error {
	q := `UPDATE things SET name = $1, metadata = $2 WHERE owner = $3 AND id = $4;`

	res, err := tr.db.Exec(q, thing.Name, thing.Metadata, thing.Owner, thing.ID)
	if err != nil {
		return err
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if cnt == 0 {
		return things.ErrNotFound
	}

	return nil
}

func (tr thingRepository) RetrieveByID(owner string, id uint64) (things.Thing, error) {
	q := `SELECT name, type, key, metadata FROM things WHERE id = $1 AND owner = $2`
	thing := things.Thing{ID: id, Owner: owner}
	err := tr.db.
		QueryRow(q, id, owner).
		Scan(&thing.Name, &thing.Type, &thing.Key, &thing.Metadata)

	if err != nil {
		empty := things.Thing{}
		if err == sql.ErrNoRows {
			return empty, things.ErrNotFound
		}
		return empty, err
	}

	return thing, nil
}

func (tr thingRepository) RetrieveByKey(key string) (uint64, error) {
	q := `SELECT id FROM things WHERE key = $1`
	var id uint64
	if err := tr.db.QueryRow(q, key).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return 0, things.ErrNotFound
		}
		return 0, err
	}

	return id, nil
}

func (tr thingRepository) RetrieveAll(owner string, offset, limit int) []things.Thing {
	q := `SELECT id, name, type, key, metadata FROM things WHERE owner = $1 ORDER BY id LIMIT $2 OFFSET $3`
	items := []things.Thing{}

	rows, err := tr.db.Query(q, owner, limit, offset)
	if err != nil {
		tr.log.Error(fmt.Sprintf("Failed to retrieve things due to %s", err))
		return []things.Thing{}
	}
	defer rows.Close()

	for rows.Next() {
		c := things.Thing{Owner: owner}
		if err = rows.Scan(&c.ID, &c.Name, &c.Type, &c.Key, &c.Metadata); err != nil {
			tr.log.Error(fmt.Sprintf("Failed to read retrieved thing due to %s", err))
			return []things.Thing{}
		}
		items = append(items, c)
	}

	return items
}

func (tr thingRepository) Remove(owner string, id uint64) error {
	q := `DELETE FROM things WHERE id = $1 AND owner = $2`
	tr.db.Exec(q, id, owner)
	return nil
}
