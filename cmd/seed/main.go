// Command seed fills the database with restaurants in Bengaluru, their dishes,
// two customers, and one staff member for each restaurant.
// It empties every table first, so run it only on your own machine.
//
// For load tests, ask for more restaurants. The ones after the first 20 get
// made-up names, a random spot in Bengaluru, and -dishes dishes each:
//
//	go run ./cmd/seed -restaurants 20000 -dishes 100   # 2 million dishes
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
)

type restaurant struct {
	name, address string
	lat, lng      float64
	dishes        []dish
}

type dish struct {
	name  string
	price int // in paise
}

var restaurants = []restaurant{
	{"Biryani House", "80 Feet Road, Koramangala", 12.9352, 77.6245, []dish{
		{"Chicken biryani", 25000}, {"Mutton biryani", 32000}, {"Veg biryani", 18000}, {"Raita", 4000}, {"Double ka meetha", 9000}}},
	{"Udupi Tiffin Room", "100 Feet Road, Indiranagar", 12.9719, 77.6412, []dish{
		{"Masala dosa", 9000}, {"Idli vada", 7000}, {"Rava idli", 8000}, {"Filter coffee", 3500}, {"Kesari bath", 5000}}},
	{"Dosa Corner", "11th Main, Jayanagar 4th Block", 12.9250, 77.5938, []dish{
		{"Benne dosa", 8500}, {"Set dosa", 7500}, {"Onion uttapam", 9000}, {"Pongal", 7000}}},
	{"Punjabi Dhaba", "27th Main, HSR Layout", 12.9116, 77.6474, []dish{
		{"Butter chicken", 29000}, {"Dal makhani", 21000}, {"Paneer tikka", 24000}, {"Butter naan", 5000}, {"Lassi", 6000}}},
	{"Chettinad Kitchen", "16th Main, BTM Layout", 12.9166, 77.6101, []dish{
		{"Chicken chettinad", 27000}, {"Kothu parotta", 16000}, {"Mutton sukka", 31000}, {"Parotta", 4000}}},
	{"Kerala Kitchen", "Outer Ring Road, Marathahalli", 12.9569, 77.7011, []dish{
		{"Appam with stew", 17000}, {"Fish moilee", 29000}, {"Kerala parotta", 4500}, {"Beef fry", 26000}}},
	{"Momo Point", "Sampige Road, Malleshwaram", 13.0031, 77.5643, []dish{
		{"Steamed chicken momos", 14000}, {"Fried veg momos", 12000}, {"Thukpa", 16000}}},
	{"Rolls and Wraps", "Brigade Road, near MG Road", 12.9756, 77.6066, []dish{
		{"Chicken kathi roll", 15000}, {"Paneer roll", 13000}, {"Egg roll", 10000}}},
	{"Andhra Meals", "Hosur Road, Electronic City", 12.8452, 77.6602, []dish{
		{"Andhra meals", 19000}, {"Gongura chicken", 26000}, {"Pesarattu", 9000}}},
	{"Thali Ghar", "ITPL Main Road, Whitefield", 12.9698, 77.7500, []dish{
		{"Rajasthani thali", 28000}, {"Mini thali", 18000}, {"Gulab jamun", 6000}}},
	{"Pizza Oven", "5th Block, Koramangala", 12.9345, 77.6190, []dish{
		{"Margherita pizza", 22000}, {"Paneer tikka pizza", 30000}, {"Garlic bread", 12000}}},
	{"Chaat Street", "Gandhi Bazaar, Basavanagudi", 12.9422, 77.5755, []dish{
		{"Pani puri", 6000}, {"Masala puri", 7000}, {"Dahi puri", 8000}, {"Bhel puri", 7000}}},
	{"Burger Shack", "12th Main, Indiranagar", 12.9700, 77.6380, []dish{
		{"Chicken burger", 19000}, {"Veg burger", 15000}, {"Peri peri fries", 11000}}},
	{"Mangalore Fish Curry", "Richmond Road, Richmond Town", 12.9650, 77.6000, []dish{
		{"Fish curry meals", 24000}, {"Neer dosa", 8000}, {"Chicken ghee roast", 32000}}},
	{"Hyderabadi Haleem", "Mosque Road, Frazer Town", 12.9980, 77.6150, []dish{
		{"Mutton haleem", 28000}, {"Chicken 65", 22000}, {"Sheer khurma", 9000}}},
	{"Bengali Bhojan", "Sarjapur Road, Bellandur", 12.9260, 77.6760, []dish{
		{"Kosha mangsho", 34000}, {"Fish fry", 21000}, {"Luchi", 7000}, {"Mishti doi", 6000}}},
	{"Gujarati Rasoi", "Dr Rajkumar Road, Rajajinagar", 12.9910, 77.5550, []dish{
		{"Gujarati thali", 26000}, {"Dhokla", 8000}, {"Khandvi", 9000}}},
	{"Idli Express", "BSK 2nd Stage, Banashankari", 12.9255, 77.5468, []dish{
		{"Thatte idli", 6000}, {"Mini idli sambar", 7000}, {"Vada", 4000}}},
	{"Tandoor Nights", "Bellary Road, Hebbal", 13.0358, 77.5970, []dish{
		{"Tandoori chicken", 30000}, {"Chicken tikka", 26000}, {"Tandoori roti", 3000}}},
	{"Mithai Mahal", "Main Road, Yelahanka New Town", 13.1007, 77.5963, []dish{
		{"Kaju katli, 250 g", 30000}, {"Rasmalai", 9000}, {"Jalebi", 6000}}},
}

func main() {
	total := flag.Int("restaurants", len(restaurants), "how many restaurants, at least 20")
	dishes := flag.Int("dishes", 100, "dishes for each restaurant after the first 20")
	flag.Parse()
	if *total < len(restaurants) || *dishes < 1 {
		fmt.Fprintf(os.Stderr, "seed: -restaurants must be at least %d, and -dishes at least 1\n", len(restaurants))
		os.Exit(2)
	}

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://food:food@localhost:5440/food?sslmode=disable"
	}
	if err := seed(context.Background(), url, *total, *dishes); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}

func seed(ctx context.Context, url string, total, dishesEach int) error {
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	return pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `TRUNCATE order_items, orders, dishes, addresses, users, restaurants RESTART IDENTITY`)
		if err != nil {
			return err
		}

		for i, r := range restaurants {
			var id int64
			err := tx.QueryRow(ctx,
				`INSERT INTO restaurants (name, address, lat, lng) VALUES ($1, $2, $3, $4) RETURNING id`,
				r.name, r.address, r.lat, r.lng,
			).Scan(&id)
			if err != nil {
				return err
			}
			for _, d := range r.dishes {
				_, err := tx.Exec(ctx,
					`INSERT INTO dishes (restaurant_id, name, price) VALUES ($1, $2, $3)`,
					id, d.name, d.price,
				)
				if err != nil {
					return err
				}
			}
			// Restaurant 1's staff log in with +919810000001, restaurant 2's with +919810000002, and so on.
			_, err = tx.Exec(ctx,
				`INSERT INTO users (name, phone, role, restaurant_id) VALUES ($1, $2, 'restaurant', $3)`,
				r.name+" manager", fmt.Sprintf("+9198100000%02d", i+1), id,
			)
			if err != nil {
				return err
			}
		}

		customers := []struct{ name, phone, address string }{
			{"Asha", "+919800000001", "Flat 302, 4th Cross, Koramangala 6th Block"},
			{"Vikram", "+919800000002", "12, 2nd Main, Indiranagar"},
		}
		for _, c := range customers {
			var id int64
			err := tx.QueryRow(ctx,
				`INSERT INTO users (name, phone, role) VALUES ($1, $2, 'customer') RETURNING id`,
				c.name, c.phone,
			).Scan(&id)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `INSERT INTO addresses (user_id, address) VALUES ($1, $2)`, id, c.address)
			if err != nil {
				return err
			}
		}

		fmt.Printf("seeded %d restaurants, their dishes and staff, and %d customers\n", len(restaurants), len(customers))

		if total > len(restaurants) {
			return seedMany(ctx, tx, total, dishesEach)
		}
		return nil
	})
}

// seedMany adds made-up restaurants until there are total, each with
// dishesEach dishes. Postgres makes the rows itself with generate_series,
// so even millions of dishes take seconds, not minutes.
func seedMany(ctx context.Context, tx pgx.Tx, total, dishesEach int) error {
	start := time.Now()
	first := len(restaurants) + 1

	// A random spot in a box around Bengaluru.
	_, err := tx.Exec(ctx, `
		INSERT INTO restaurants (name, address, lat, lng)
		SELECT 'Restaurant ' || g, 'Somewhere in Bengaluru', 12.85 + random() * 0.25, 77.50 + random() * 0.25
		FROM generate_series($1::int, $2::int) g`,
		first, total,
	)
	if err != nil {
		return err
	}

	// Prices from ₹50 to ₹400, in paise.
	_, err = tx.Exec(ctx, `
		INSERT INTO dishes (restaurant_id, name, price)
		SELECT r.id, 'Dish ' || g, 5000 + floor(random() * 35000)::int
		FROM restaurants r, generate_series(1, $1::int) g
		WHERE r.id >= $2`,
		dishesEach, first,
	)
	if err != nil {
		return err
	}

	fmt.Printf("added %d more restaurants with %d dishes each in %s\n",
		total-len(restaurants), dishesEach, time.Since(start).Round(time.Millisecond))
	return nil
}
