# Decisions

This app follows the design from the course. The requests come from the API lesson, and the tables come from the data modeling lesson. When we wrote the code, we found a few places where the lessons didn't say enough to write it. This file lists each one, with what we chose and why.

The rule we follow: the lessons win. If the code needs something the lessons don't have, we pick the simplest thing the lessons already point to, or we change the lesson. The code never quietly goes its own way.

## How does the server know which restaurants are near the customer?

The request is `GET /v1/restaurants?near=…`, but the restaurants table had only a name and an address. The server can't measure distance from words like "80 Feet Road, Koramangala".

So each restaurant also gets `lat` and `lng`, its spot on the map, next to its address. The address is for people to read, and lat and lng are for the server to do the maths. The phone sends its own location, like `near=12.9352,77.6245`, and the server returns the restaurants within 7 km, nearest first. You can change the 7 km with `NEAR_RADIUS_KM`.

The server works out the distance to every restaurant, so Postgres reads the whole restaurants table. With a few hundred restaurants, that's fine. The indexes lesson says this lookup needs a different kind of index once there are many restaurants, and we'll add it then.

## How does the restaurant's tablet find out about new orders?

The indexes lesson shows the tablet asking for the orders it has received and still needs to cook, and adds an index on orders `(restaurant_id, status)` for it. But no request in the API lesson did that. So we added one:

```http
GET /v1/restaurants/42/orders?status=paid
```

Only the staff of restaurant 42 can call it. The tablet asks every few seconds. Later in the course, the server will push new orders to the tablet instead.

## Where does a user's name come from?

`GET /v1/me` returns the user's name, but no request sends one. A customer logs in with only their phone number and a code. So the first time a phone number logs in, the server creates a customer with no name, and `GET /v1/me` returns `"name": null`. The seeded users have names.

No step of the customer's journey asks for a name, so we didn't add a request for it. When a screen needs one, we'll add `PATCH /v1/me`.

## What token does a user get when they log in?

`POST /v1/login/verify` returns a JWT, signed with the server's secret key (`JWT_SECRET`). It says who the user is, their role, and for restaurant staff, their restaurant. It lasts 30 days.

The server doesn't store tokens anywhere, so there's no sessions table. The signature alone proves we issued the token. The cost is that we can't cancel a token before it expires, for example when someone loses their phone. If we need that later, we'll add a table of cancelled tokens or switch to stored sessions.

## Which users can do what?

There are two roles, as in the data modeling lesson: customer and restaurant. Customers use the customer requests, and restaurant staff use the tablet requests. Staff can change only their own restaurant's orders and dishes.

When a customer asks for someone else's order, the server answers 404, not 403. So nobody can find out which order IDs exist by trying them.

## How does the payment webhook work?

The requirements lesson names the request `POST /payments/webhook`. It has no `/v1`, because the payment provider decides what it sends, not our API.

Anyone on the internet can call that URL, so the server checks each webhook really came from the provider. The provider signs the body with a secret that only it and our server know (`PAYMENT_WEBHOOK_SECRET`), and sends the signature in the `X-Signature` header. Razorpay signs its webhooks the same way. A webhook with a wrong signature gets a 401 and changes nothing.

## What happens to an order's status?

An order moves through `pending → paying → paid → accepted → ready`. Every move is one SQL statement with the old status in its WHERE, like this:

```sql
UPDATE orders SET status = 'paying' WHERE id = $1 AND status = 'pending'
```

If the customer taps **Pay** twice, both requests try this update, but only one of them finds the order still pending. The other changes no rows, so the server answers 409 and doesn't charge again.

Two more moves come from the payment provider. A captured payment moves the order from paying to paid. A failed payment moves it from paying back to pending, so the customer can try again. Providers can send the same webhook twice, so a webhook for an order that's no longer paying changes nothing, and the server still answers 200. If it answered with an error, the provider would keep sending it.

## What does an order keep from the moment it was placed?

The data modeling lesson asks: if it changes, should old records show their old value? For an order, yes. So the order keeps its own copy of the delivery address text, and each item keeps the dish's price at the time of the order. When a restaurant changes a price, or the customer edits their address, orders already placed don't change.

## Why doesn't order_items have a primary key?

The data model in the course doesn't give order_items one, and the indexes lesson adds an index on `order_items (order_id)`. A primary key on `(order_id, dish_id)` would make that index unnecessary, so the code and the lessons would disagree.

The server makes sure the same dish appears only once in an order: if the app sends a dish twice, the server answers 400 and asks for one line with the quantity.

## Why are the OTP service and the payment provider fake?

A real OTP service sends real text messages, and in India that needs DLT registration first. A real payment provider needs an account and a public URL for its webhook, and someone has to approve each test payment by hand. Neither works for a project you clone and run in five minutes, and neither works for load tests: we want to measure our server, not someone else's.

So the server talks to both through a small Go interface, and this repo has a fake for each:

- The fake OTP service accepts the code `123456` for every phone, and logs it instead of texting it.
- The fake payment provider answers at once with a payment ID. Then, 1 to 2 seconds later, as if the customer typed their UPI PIN, it calls our webhook with a signed "captured". With `FAKEPAY_FAIL_RATE` and `FAKEPAY_DUPLICATE_RATE`, it fails some payments and sends some webhooks twice, like a real provider.

A real provider can replace either fake later without changing the services.

## Money

All amounts are whole numbers of paise, so ₹250 is `25000`, in the database and in the API. The lessons write prices in rupees to keep them easy to read.
