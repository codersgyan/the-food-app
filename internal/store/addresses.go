package store

import "context"

type Address struct {
	ID      int64
	Address string
}

// AddressesOf returns a user's saved addresses, oldest first.
func (s *Store) AddressesOf(ctx context.Context, userID int64) ([]Address, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, address FROM addresses WHERE user_id = $1 ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Address
	for rows.Next() {
		var a Address
		if err := rows.Scan(&a.ID, &a.Address); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}
