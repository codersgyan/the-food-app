package store

import "context"

type Restaurant struct {
	ID         int64
	Name       string
	Address    string
	DistanceKm float64 // set only by RestaurantsNear
}

// RestaurantsNear returns the restaurants within radiusKm of a spot on the
// map, nearest first, limit at a time, skipping the first offset.
//
// ST_MakePoint(lng, lat)::geography is the same expression as the spatial
// index in migration 00011, and <-> is the distance in metres. Ordering by it
// lets Postgres walk the index nearest first and stop after limit rows. The
// radius is checked outside, on those few rows. With the radius inside, as
// ST_DWithin, Postgres reads every restaurant in the circle and sorts them,
// which is slower than no index at all when the circle covers a big part of
// the city.
func (s *Store) RestaurantsNear(ctx context.Context, lat, lng, radiusKm float64, limit, offset int) ([]Restaurant, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, address, meters / 1000
		FROM (
			SELECT id, name, address,
			       ST_MakePoint(lng, lat)::geography <-> ST_MakePoint($2::float8, $1::float8)::geography AS meters
			FROM restaurants
			ORDER BY ST_MakePoint(lng, lat)::geography <-> ST_MakePoint($2::float8, $1::float8)::geography
			LIMIT $4 OFFSET $5
		) nearest
		WHERE meters <= $3::float8 * 1000`,
		lat, lng, radiusKm, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Restaurant
	for rows.Next() {
		var r Restaurant
		if err := rows.Scan(&r.ID, &r.Name, &r.Address, &r.DistanceKm); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (s *Store) Restaurant(ctx context.Context, id int64) (Restaurant, error) {
	var r Restaurant
	err := s.db.QueryRow(ctx,
		`SELECT id, name, address FROM restaurants WHERE id = $1`, id,
	).Scan(&r.ID, &r.Name, &r.Address)
	return r, notFound(err)
}
