package store

import "context"

type Restaurant struct {
	ID         int64
	Name       string
	Address    string
	DistanceKm float64 // set only by RestaurantsNear
}

// RestaurantsNear returns the restaurants within radiusKm of a spot on the
// map, nearest first.
//
// The distance is worked out for every restaurant, with the haversine
// formula for distance on a sphere (6371 km is the Earth's radius). That
// means reading the whole table, which is fine while there are a few hundred
// restaurants. When there are many, this needs a geo index.
func (s *Store) RestaurantsNear(ctx context.Context, lat, lng, radiusKm float64) ([]Restaurant, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, address, distance_km
		FROM (
			SELECT id, name, address,
			       6371 * 2 * asin(sqrt(
			           power(sin(radians(lat - $1) / 2), 2) +
			           cos(radians($1)) * cos(radians(lat)) * power(sin(radians(lng - $2) / 2), 2)
			       )) AS distance_km
			FROM restaurants
		) r
		WHERE distance_km <= $3
		ORDER BY distance_km`,
		lat, lng, radiusKm,
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
