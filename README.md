# [portal]

Tiny API and Agent server enabling AI models to access various local services in the real world.

<div align="center">
  <img src="docs/be0c85ba-e22b-442f-827c-ac2b9a430b49.jpg" alt="Project picture" style="width: 50%; border-radius: 15px;" />
</div>

> [!WARNING]  
> The project is in the early stages of development and is not ready for production use.

## Features

- [x] [Docker support](#docker)
- [x] [Webhooks](#webhooks)
- [x] [WebSockets](#websockets)
- [x] [Date and time API](#date-and-time)
- [x] [Geolocation API](#geolocation-api)
- [x] [DuckDuckGo Instant Answers API](#duckduckgo-instant-answers-api)
- [x] [Reminders API](#reminders-api)
- [x] [Chats API](#chats-api)
- [ ] Notes API
- [ ] Web Search API
- [ ] Weather API
- [ ] File System API
- [ ] Plugins API

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

## Building from Source

To build the server binary:

```shell
go build -o portal .
```

To run the server binary:

```shell
./portal
```

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

The server provides a simple REST API for accessing various services.

All API endpoints are prefixed with `/api`.

### Date and Time

- [GET /date.now](#get-datenow)

#### GET /date.now

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

### Geolocation API

- [GET /geolocation](#get-geolocation)

#### GET /geolocation

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

### DuckDuckGo Instant Answers API

- [GET /search.instant?q={query}](#get-searchinstantqquery)

#### GET /search.instant?q={query}

Returns an instant answer for the given search query.

Uses the DuckDuckGo Instant Answer API:
`https://api.duckduckgo.com/?q=hello&format=json`

Examples:

- `/search.instant?q=global+warming`
- `/search.instant?q=hello%20world`

### Reminders API

Provides a simple API for creating and managing reminders.
Reminders can be scheduled for a specific date and time.

You can also specify a webhook URL to send reminders to external services.
WebSockets are used to notify clients about new reminders.

- [GET /reminders.list](#get-reminderslist)
- [POST /reminders.add](#post-remindersadd)
- [POST /reminders.complete](#post-reminderscomplete)
- [POST /reminders.delete](#post-remindersdelete)
- [GET /reminders.info](#get-remindersinfo)

#### GET /reminders.list

Returns a list of reminders.

Example:

```shell
curl -X GET "http://localhost:1323/api/reminders.list"
```

```json
[
  {
    "id": "71f4491c-417e-4fe0-aa06-5722b942e273",
    "message": "Buy milk",
    "description": "Buy 2% milk",
    "due_time": "2025-03-18T21:08:28-04:00",
    "completed": false
  },
  {
    "id": "09d27d7f-27ff-4e01-9598-104b3d654675",
    "title": "Call mom",
    "description": "Call mom on her birthday",
    "due_time": "2025-03-18T21:08:28-04:00",
    "completed": false
  }
]
```

#### POST /reminders.add

Creates a new reminder.

Request body:

- `message`: The reminder message
- `description`: The reminder description
- `due_time`: The due date and time in RFC3339 format
- `webhook_url`: The URL of the webhook receiver (optional)

Example:

```shell
curl -X POST "http://localhost:1323/api/reminders.add" \
  -H "Content-Type: application/json" \
  -d '{
      "message": "Buy milk",
      "description": "Buy 2% milk",
      "due_time":"'"$(date -v +2M +"%Y-%m-%dT%H:%M:%SZ")"'"
    }'
```

The returned reminder object:

```json
{
  "id": "09d27d7f-27ff-4e01-9598-104b3d654675",
  "message": "Buy milk",
  "description": "Buy 2% milk",
  "due_time": "2025-03-18T21:08:28-04:00",
  "completed": false,
  "webhook_url": "<webhook_url>"
}
```

The Reminders Agent has a built-in scheduler with precision up to the minute.
It uses the `due_time` field to schedule the reminder.

#### POST /reminders.complete

Marks a reminder as completed.

Parameters:

- `id`: The reminder ID

Example:

```shell
curl -X POST "http://localhost:1323/api/reminders.complete?id=123"
```

#### POST /reminders.delete

Deletes a reminder.

Parameters:

- `id`: The reminder ID

Example:

```shell
curl -X POST "http://localhost:1323/api/reminders.delete?id=123"
```

#### GET /reminders.info

Returns information about a specific reminder.

Parameters:

- `id`: The reminder ID

Example:

```shell
curl -X GET "http://localhost:1323/api/reminders.info?id=09d27d7f-27ff-4e01-9598-104b3d654675"
```

```json
{
  "id": "09d27d7f-27ff-4e01-9598-104b3d654675",
  "message": "Buy milk",
  "description": "Buy 2% milk",
  "due_time": "2025-03-18T21:08:28-04:00",
  "completed": false
}
```

#### Webhooks

The reminders API supports webhooks for sending reminders to external services.

When creating a new reminder, you can specify a `webhook_url` field with the URL of the webhook receiver.

Example of creating a new reminder with a webhook, running 2 minutes from now:

```shell
curl -X POST http://localhost:1323/api/reminders.add \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Test reminder",
    "description":"This is a test reminder",
    "due_time":"'"$(date -v +2M +"%Y-%m-%dT%H:%M:%SZ")"'",
    "webhook_url":"http://your-webhook-receiver/webhook"
  }'
```

### Chats API

- [POST /chats.add](#post-chatsadd)
- [GET /chats.list](#get-chatslist)
- [GET /chats.info](#get-chatsinfo)
- [POST /chats.delete](#post-chatsdelete)
- [POST /chats.rename](#post-chatsrename)
- [POST /chats.pin](#post-chatspin)
- [POST /chats.unpin](#post-chatsunpin)
- [POST /messages.add](#post-messagesadd)
- [GET /messages.list](#get-messageslist)

#### POST /chats.add

Creates a new chat.

Request body:

- `user_id`: The user ID
- `title`: The chat title
- `timestamp`: The chat creation timestamp

Example:

```shell
curl -X POST "http://localhost:1323/api/chats.add" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "some-user-id",
    "title": "My Chat"
  }'
````

The returned chat object:

```json
{
  "id": "d6924d7f-e53d-452e-83a0-0f0893de68b5",
  "user_id": "some-user-id",
  "title": "My Chat",
  "timestamp": 1742551200
}
```

#### GET /chats.list

Returns a list of chats.

Parameters:

- `user_id`: The user ID

Example:

```shell
curl -X GET "http://localhost:1323/api/chats.list?user_id=some-user-id"
```

```json
[
  {
    "id": "d6924d7f-e53d-452e-83a0-0f0893de68b5",
    "user_id": "some-user-id",
    "title": "My Chat",
    "timestamp": 1742551200,
    "is_pinned": false
  }
]
```

#### GET /chats.info

Returns information about a specific chat.

Parameters:

- `chat_id`: The chat ID
- `user_id`: The user ID

Example:

```shell
curl -X GET "http://localhost:1323/api/chats.info?chat_id=d6924d7f-e53d-452e-83a0-0f0893de68b5&user_id=some-user-id"
```

Response:

```json
{
  "id": "d6924d7f-e53d-452e-83a0-0f0893de68b5",
  "user_id": "some-user-id",
  "title": "My Chat",
  "timestamp": 1742551200,
  "is_pinned": false
}
```

#### POST /chats.delete

Deletes a chat.

Request body:

- `user_id`: The user ID
- `chat_id`: The chat ID

Example:

```shell
curl -X POST "http://localhost:1323/api/chats.delete" \
  -H "Content-Type: application/json" \
  -d '{
	"chat_id": "d6924d7f-e53d-452e-83a0-0f0893de68b5",
	"user_id": "some-user-id"
  }'
```

#### POST /chats.rename

Renames a chat.

Request body:

- `user_id`: The user ID
- `chat_id`: The chat ID
- `title`: The new chat title

Example:

```shell
curl -X POST "http://localhost:1323/api/chats.rename" \
  -H "Content-Type: application/json" \
  -d '{
	"chat_id": "d6924d7f-e53d-452e-83a0-0f0893de68b5",
	"user_id": "some-user-id",
	"title": "New Chat Title"
  }'
```

#### POST /chats.pin

Pins a chat.

Request body:

- `chat_id`: The chat ID
- `user_id`: The user ID

Example:

```shell
curl -X POST "http://localhost:1323/api/chats.pin" \
  -H "Content-Type: application/json" \
  -d '{
    "chat_id": "d6924d7f-e53d-452e-83a0-0f0893de68b5",
    "user_id": "some-user-id"
  }'
```

#### POST /chats.unpin

Unpins a chat.

Request body:

- `chat_id`: The chat ID
- `user_id`: The user ID

Example:

```shell
curl -X POST "http://localhost:1323/api/chats.unpin" \
  -H "Content-Type: application/json" \
  -d '{
    "chat_id": "d6924d7f-e53d-452e-83a0-0f0893de68b5",
    "user_id": "some-user-id"
  }'
```

#### POST /messages.add

Adds a new message to a chat.

Request body:

- `chat_id`: The chat ID
- `sender`: The sender user or model ID in the format `user:<user-id>` or `model:<model-id>`
- `sender_role`: The sender role (`user` | `assistant` | `tool` | `system`)
- `content`: The message content, text or JSON
- `timestamp`: The message timestamp in Unix time format
- `tools`: The list of tools used in the message, in JSON format (optional)

Example:

```shell
curl -X POST "http://localhost:1323/api/messages.add" \
  -H "Content-Type: application/json" \
  -d '{
    "chat_id": "d6924d7f-e53d-452e-83a0-0f0893de68b5",
    "sender": "user:some-user",
    "sender_role": "user",
    "content": "Hello, world!"
  }'
```

The returned message object:

```json
{
  "id": "c4de2af4-ea23-45a1-b039-cadace10491f",
  "chat_id": "d6924d7f-e53d-452e-83a0-0f0893de68b5",
  "sender": "user:some-user",
  "sender_role": "user",
  "content": "Hello, world!",
  "timestamp": 1742551200,
  "feedback": 0,
  "tools": null
}
```

#### GET /messages.list

Returns a list of messages in a chat.

Parameters:

- `chat_id`: The chat ID

Example:

```shell
curl -X GET "http://localhost:1323/api/messages.list?chat_id=d6924d7f-e53d-452e-83a0-0f0893de68b5"
```

```json
[
  {
    "id": "c4de2af4-ea23-45a1-b039-cadace10491f",
    "chat_id": "d6924d7f-e53d-452e-83a0-0f0893de68b5",
    "sender": "user:some-user",
    "sender_role": "user",
    "content": "Hello, world!",
    "timestamp": 1742551200,
  }
]
```

## WebSockets

The server supports WebSockets for real-time communication with clients.

- [GET /ws](#ws)

Channels:

- `api.reminders`: Receive reminders in real-time

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

### Building the Project

To build the project:

```shell
go build -ldflags "-X main.Version=1.0.0 -X main.BuildDate=$(date -u +%Y-%m-%d) -X main.GitCommit=$(git rev-parse --short HEAD)" -o portal
```

Where Version, BuildDate, and GitCommit are set as build-time variables.

### Database Migrations

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
