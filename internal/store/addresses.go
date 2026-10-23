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

// AddAddress saves an address for a user and returns its ID.
func (s *Store) AddAddress(ctx context.Context, userID int64, address string) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO addresses (user_id, address) VALUES ($1, $2) RETURNING id`, userID, address,
	).Scan(&id)
	return id, err
}

// Address returns one of the user's saved addresses.
func (s *Store) Address(ctx context.Context, id, userID int64) (Address, error) {
	var a Address
	err := s.db.QueryRow(ctx,
		`SELECT id, address FROM addresses WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&a.ID, &a.Address)
	return a, notFound(err)
}
