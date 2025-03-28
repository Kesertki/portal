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
- [x] [Files API](#files-api)
- [x] [Storage API (S3 compatible)](#storage-api-s3-compatible)
- [ ] Notes API
- [ ] Web Search API
- [ ] Weather API
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

The server can be run in a Docker container. The Docker image is available on Docker Hub.

To pull the Docker image:

```shell
docker pull kesertki/portal
```

### Building and Running Docker Container

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

### Running with Docker Compose

Create a `docker-compose.yml` file:

```yaml
version: '3.8'
services:
  api:
    container_name: api
    image: kesertki/portal:latest
    ports:
      - "1323:1323"
    # environment:
    #   PORTAL_GEO_LOCATION_ENABLED: true
    #   PORTAL_CLIENT_IP: "1.1.1.1"
    env_file:
      - .env.prod
    volumes:
      - ./data:/root/data
```

To run the Docker container with Docker Compose:

```shell
docker-compose up -d
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

### Files API

The Files API provides a simple way to upload and download files.

- [POST /fs/files](#post-fsfiles)
- [GET /fs/files/*](#get-fsfiles)
- [PUT /fs/files/*](#put-fsfiles)
- [DELETE /fs/files/*](#delete-fsfiles)
- [GET /list/*](#get-list)

#### POST /fs/files

Uploads a file.

Request body:

- `file`: The file to upload
- `user_id`: The user ID
- `path`: The path to save the file

Example:

```shell
curl -X POST http://localhost:1323/api/fs/files \
     -F "file=@./data/fs/README.md" \
     -F "user_id=123e4567-e89b-12d3-a456-426614174000" \
     -F "path=/user/files"
```

#### GET /fs/files/*

Downloads a file.

Everything after `/files/` is treated as the file path.

Query parameters:

- `user_id`: The user ID

Example:

```shell
# View the file content
curl -X GET "http://localhost:1323/api/fs/files/user/files/README.md?user_id=123e4567-e89b-12d3-a456-426614174000"

# Save the file to disk
curl -X GET "http://localhost:1323/api/fs/files/user/files/README.md?user_id=123e4567-e89b-12d3-a456-426614174000" \
    --output data/file.md
```

#### PUT /fs/files/*

Updates a file.

Everything after `/files/` is treated as the file path.

Request body:

- `file`: The file to upload
- `user_id`: The user ID

Example:

```shell
curl -X PUT "http://localhost:1323/api/fs/files/user/files/README.md" \
     -F "file=@./data/fs/Updated.md" \
     -F "user_id=123e4567-e89b-12d3-a456-426614174000"
```

#### DELETE /fs/files/*

Deletes a file.

Everything after `/files/` is treated as the file path.

Query parameters:

- `user_id`: The user ID

Example:

```shell
curl -X DELETE "http://localhost:1323/files/user/files/README.md?user_id=123e4567-e89b-12d3-a456-426614174000"
```

#### GET /list/*

Lists files in a directory.

Everything after `/files/` is treated as the directory path.

Query parameters:

- `user_id`: The user ID

Example:

```shell
curl -X GET "http://localhost:1323/api/fs/list/user/files?user_id=123e4567-e89b-12d3-a456-426614174000"
```

```json
[
  "README.md"
]
```

### Storage API (S3 compatible)

Provides a simple S3-compatible storage API for uploading and downloading files. Basic compatibility with `s3cmd` and other S3 clients.

> All api endpoints are prefixed with `/api/storage`.

- `GET /` - List all buckets
- `POST /buckets/:bucket` - Create a new bucket
- `DELETE /buckets/:bucket` - Delete a bucket
- `PUT /buckets/:bucket/objects/:key` - Upload object
- `POST /buckets/:bucket/objects/:key` - Initiate a multipart upload
- `PUT /buckets/:bucket/objects/:key/uploads` - Upload a part of the multipart object
- `POST /buckets/:bucket/objects/:key/complete` - Complete the multipart upload
- `DELETE /buckets/:bucket/objects/:key/uploads` - Abort the multipart upload
- `GET /buckets/:bucket/objects/:key` - Download an object
- `HEAD /buckets/:bucket/objects/:key` - Get object metadata
- `GET /buckets/:bucket/objects` - List all objects in a bucket
- `DELETE /buckets/:bucket/objects/:key` - Delete an object
- `GET /:bucket` - List all objects in a bucket
- `PUT /:bucket` - Create a new bucket
- `GET /:bucket/:key` - Download an object
- `HEAD /:bucket/:key` - Get object metadata
- `DELETE /:bucket` - Delete a bucket
- `DELETE /:bucket/:key` - Delete an object

#### Using with CURL

Create a new bucket:

```shell
curl -X POST http://localhost:1323/api/storage/buckets/mybucket
```

Response:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<CreateBucketResponse>
	<Message>Bucket created successfully</Message>
</CreateBucketResponse>
```

Upload a new object:

```shell
curl -X PUT \
     -F "file=@README.md;type=text/markdown" \
     http://localhost:1323/api/storage/buckets/mybucket/objects/README.md
```

Response:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<PutObjectResult>
	<ETag>5520927df1b08a8f5778c03b21c75d64</ETag>
</PutObjectResult>
```

Download an object:

```shell
curl -X GET http://localhost:1323/api/storage/buckets/mybucket/objects/README.md \
    --output downloaded.md
```

List all objects in a bucket:

```shell
curl -X GET http://localhost:1323/api/storage/buckets/mybucket/objects
```

Response example:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult>
    <Name>mybucket</Name>
    <Prefix></Prefix>
    <Marker></Marker>
    <MaxKeys>1000</MaxKeys>
    <IsTruncated>false</IsTruncated>
    <Contents>
        <Key>README.md</Key>
        <LastModified>2025-03-27T22:05:28Z</LastModified>
        <ETag>312a0794bb855b7cf9cda79422871489</ETag>
        <Size>20007</Size>
        <StorageClass>STANDARD</StorageClass>
    </Contents>
    <CommonPrefixes></CommonPrefixes>
</ListBucketResult>
```

Delete an object:

```shell
curl -X DELETE http://localhost:1323/api/storage/buckets/mybucket/objects/README.md
```

#### Multi-part Upload

**Step 1: Initiate Multipart Upload**:

Initiate a multipart upload by sending a POST request to your server. This will return an uploadId that you will use in subsequent requests.

```shell
curl -X POST http://localhost:1323/api/storage/buckets/mybucket/objects/myobject
```

**Step 2: Upload Parts**:

Upload individual parts of the object using the uploadId obtained from the previous step. You need to specify the partNumber and uploadId as query parameters.

```shell
curl -X PUT "http://localhost:1323/api/storage/buckets/mybucket/objects/myobject/uploads?partNumber=1&uploadId=your-upload-id" \
     -H "Content-Type: application/octet-stream" \
     --data-binary @part1.txt

curl -X PUT "http://localhost:1323/api/storage/buckets/mybucket/objects/myobject/uploads?partNumber=2&uploadId=your-upload-id" \
     -H "Content-Type: application/octet-stream" \
     --data-binary @part2.txt
```

**Step 3: Complete Multipart Upload**:

Complete the multipart upload by sending a POST request with the uploadId. You need to provide a list of parts in the request body.

```shell
curl -X POST "http://localhost:1323/api/storage/buckets/mybucket/objects/myobject/complete?uploadId=your-upload-id" \
     -H "Content-Type: application/xml" \
     --data-binary @complete.xml
```

The **complete.xml** file should contain the list of parts, like this:

```xml
<CompleteMultipartUpload>
    <Part>
        <PartNumber>1</PartNumber>
        <ETag>etag-for-part-1</ETag>
    </Part>
    <Part>
        <PartNumber>2</PartNumber>
        <ETag>etag-for-part-2</ETag>
    </Part>
</CompleteMultipartUpload>
```

**Step 4: Abort Multipart Upload (if needed)**:

If you need to abort the upload, you can send a DELETE request with the uploadId.

```shell
curl -X DELETE "http://localhost:1323/api/storage/buckets/mybucket/objects/myobject/uploads?uploadId=your-upload-id"
```

**Notes**:

- Replace mybucket, myobject, your-upload-id, part1.txt, and part2.txt with your actual bucket name, object key, upload ID, and part files.
- The ETag values in the complete.xml file should match the ETags returned by the server when you uploaded each part.

#### Using with s3cmd

Create a new S3 configuration file:

```shell
cat <<EOF > .s3cfg
[default]
access_key = your-access-key
secret_key = your-secret-key
host_base = localhost:1323/api/storage
host_bucket = localhost:1323/api/storage/%(bucket)
use_https = False
EOF
```

Create a new bucket:

```shell
s3cmd -c .s3cfg mb s3://mybucket
```

Response:

```text
Bucket 's3://mybucket/' created
```

List all buckets:

```shell
s3cmd -c .s3cfg ls      
```

Example response:

```text
2025-03-27 19:06  s3://mybucket
2025-03-27 19:12  s3://mybucket1
2025-03-27 19:18  s3://mybucket2
2025-03-27 19:19  s3://mybucket3
2025-03-27 19:26  s3://mybucket4
2025-03-27 19:26  s3://mybucket5
```

Delete the bucket:

```shell
s3cmd -c .s3cfg rb s3://mybucket5
```

Response:

```text
Bucket 's3://mybucket5/' removed
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
