package store

import "context"

type User struct {
	ID           int64
	Name         *string // nil until the app asks for a name
	Phone        string
	Role         string
	RestaurantID *int64 // set only for restaurant staff
}

// UserByPhoneOrNew returns the user with this phone. The first time a phone
// number logs in, it creates a customer for it.
func (s *Store) UserByPhoneOrNew(ctx context.Context, phone string) (User, error) {
	var u User
	// ON CONFLICT ... DO UPDATE changes nothing here. It's there so that
	// RETURNING gives back the user who already has this phone.
	err := s.db.QueryRow(ctx, `
		INSERT INTO users (phone, role) VALUES ($1, 'customer')
		ON CONFLICT (phone) DO UPDATE SET phone = EXCLUDED.phone
		RETURNING id, name, phone, role, restaurant_id`,
		phone,
	).Scan(&u.ID, &u.Name, &u.Phone, &u.Role, &u.RestaurantID)
	return u, err
}
