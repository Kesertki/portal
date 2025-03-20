# [portal]

Tiny API and Agent server enabling AI models to access various local services in the real world.

<div align="center">
  <img src="docs/be0c85ba-e22b-442f-827c-ac2b9a430b49.jpg" alt="Project picture" style="width: 50%; border-radius: 15px;" />
</div>

> [!WARNING]  
> The project is in the early stages of development and is not ready for production use.

## Features

- [x] Docker support
- [x] Webhooks
- [x] WebSockets
- [ ] Plugins API
- [x] Date and time API
- [x] Geolocation API
- [x] DuckDuckGo Instant Answers API
- [x] Reminders API
- [ ] Notes API
- [ ] Web Search API
- [ ] Weather API

## Running from Source

Create a `.env` file with the following content:

```yaml
DATA_PATH=./data
PORTAL_GEO_LOCATION_ENABLED=true
```

Run the server:

```shell
go run .
```

Environment variables:

- `DATA_PATH`: The path to the data directory (default: `.`)
- `PORTAL_GEO_LOCATION_ENABLED`: Boolean, toggle the geolocation feature (default: false)
- `PORTAL_CLIENT_IP`: The static IP address to use for geo location

## Docker

To build and run the Docker container:

```shell
# Build and run the Docker container
docker build -t portal-api .
docker run -p 1323:1323 portal-api

# Run the Docker container with environment variables
docker run --name portal-api --env-file .env.prod -p 1323:1323 portal-api
```

Example `.env.prod` file:

```yaml
PORTAL_GEO_LOCATION_ENABLED=true
PORTAL_CLIENT_IP=1.1.1.1
```

### Running Docker Compose

To run the Docker container with Docker Compose:

```shell
docker-compose up --build
```

## API

### GET /api/date.now

Returns the current date and time in RFC3339 format

Example:

```shell
curl -X GET "http://localhost:1323/api/date.now"
```

```json
{
  "date": "2025-03-18T21:08:28-04:00"
}
```

### GET /api/geolocation

Returns geolocation information for the client's IP address,
or for the IP address specified in the `X-Forwarded-For` header,
or for the IP address specified in the `PORTAL_CLIENT_IP` environment variable.

Requires `PORTAL_GEO_LOCATION_ENABLED=true`, otherwise always returns the default location:

```text
IP:       "8.8.8.8",
City:     "Mountain View",
Region:   "California",
Country:  "US",
Loc:      "37.3860,-122.0840",
Org:      "Google LLC",
Postal:   "94035",
Timezone: "America/Los_Angeles",
```

> [!NOTE]  
> Make sure that your server is configured correctly to pass the client's IP address if it's behind a proxy or load balancer.
> You might need to set the `X-Real-IP` or `X-Forwarded-For` headers appropriately in your proxy or load balancer configuration.

You can also set a fixed IP address for geolocation using `PORTAL_CLIENT_IP` environment variable:

```yaml
PORTAL_CLIENT_IP=1.1.1.1
```

The API caches the geolocation to avoid repeated requests to the IP geolocation service.

Example:

```shell
curl -X GET "http://localhost:1323/api/geolocation"
curl -H "X-Forwarded-For: 8.8.8.8" http://localhost:1323/api/geolocation
```

```json
{
 "ip": "8.8.8.8",
 "city": "Mountain View",
 "region": "California",
 "country": "US",
 "loc": "37.3860,-122.0840",
 "org": "Google LLC",
 "postal": "94035",
 "timezone": "America/Los_Angeles"
}
```

### GET /api/search.instant?q={query}

Returns an instant answer for the given search query.

Uses the DuckDuckGo Instant Answer API:
`https://api.duckduckgo.com/?q=hello&format=json`

Examples:

- `/api/search.instant?q=global+warming`
- `/api/search.instant?q=hello%20world`

### GET /api/reminders

Returns a list of reminders.

Example:

```shell
curl -X GET "http://localhost:1323/api/reminders"
```

```json
[
  {
	"id": 1,
	"message": "Buy milk",
	"description": "Buy 2% milk",
	"due_date": "2025-03-18T21:08:28-04:00",
	"completed": false
  },
  {
	"id": 2,
	"title": "Call mom",
	"description": "Call mom on her birthday",
	"due_date": "2025-03-18T21:08:28-04:00",
	"completed": false
  }
]
```

### POST /api/reminders

Creates a new reminder.

Example:

```shell
curl -X POST "http://localhost:1323/api/reminders" \
	-H "Content-Type: application/json" \
	-d "{\"message\":\"Buy milk\",\"description\":\"Buy 2% milk\",\"due_date\":\"2025-03-18T21:08:28-04:00\",\"completed\":false}"
```

```json
{
  "id": 1,
  "message": "Buy milk",
  "description": "Buy 2% milk",
  "due_date": "2025-03-18T21:08:28-04:00",
  "completed": false,
  "webhook_url": "<webhook_url>"
}
```

The Reminders Agent has a built-in scheduler with precision up to the minute.
It uses the `due_date` field to schedule the reminder.

## Webhooks

The reminders API supports webhooks for sending reminders to external services.

When creating a new reminder, you can specify a `webhook_url` field with the URL of the webhook receiver.

Example of creating a new reminder with a webhook, running 2 minutes from now:

```shell
curl -X POST http://localhost:1323/api/reminders \
-H "Content-Type: application/json" \
-d '{"message":"Test reminder","description":"This is a test reminder","due_time":"'"$(date -v +2M +"%Y-%m-%dT%H:%M:%SZ")"'","completed":false,"webhook_url":"http://your-webhook-receiver/webhook"}'
```

## WebSockets

The server supports WebSockets for real-time communication with clients.

### /ws

The WebSocket endpoint for the reminders API.

Example:

```javascript
const ws = new WebSocket('ws://localhost:1323/ws');

ws.onopen = () => {
  console.log('WebSocket connection established');
};

ws.onmessage = (event) => {
  console.log('WebSocket message received:', event.data);
};

ws.onclose = () => {
  console.log('WebSocket connection closed');
};
```

## Development

Creating new migrations:

```shell
migrate create -ext sql -dir db/migrations -seq create_users_table
```

Update the created migration file with the SQL statements.

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL
);
```

The migrations are applied automatically when the server starts.

To manually run the migrations:

```shell
# Run the migrations
migrate -path db/migrations -database "sqlite3://./data/portal.db" up

# Rollback the migrations
migrate -path db/migrations -database "sqlite3://./data/portal.db" down
```

### Troubleshooting

Installing the `migrate` tool:

```shell
go install -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
mv ~/go/bin/migrate /usr/local/bin/
migrate -help
```

Alternative way to run the migrations:

```shell
# Run the migrations
go run github.com/golang-migrate/migrate/v4/cmd/migrate@latest -path db/migrations -database "sqlite3://./data/portal.db" down

# Run the migrations using Docker
docker run --rm -v $(pwd)/db/migrations:/migrations migrate/migrate -path=/migrations -database "sqlite3://./data/portal.db" down
```
