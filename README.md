# GWI - Jedi Team - Engineering Challenge

This project implements a chatbot server in Go that answers questions based on market research data provided in a Markdown table format (`data.md`). It uses a Retrieval Augmented Generation (RAG) approach with Google's Gemini LLM and persists chat conversations in an SQLite database.

## Features

- **Go Web Server:** Exposes RESTful API endpoints for chat interactions.
- **RAG Pipeline with Gemini:**
  - Data from `data.md` (table format, one fact per row) is chunked (each row is a chunk).
  - Chunks are embedded using Google's `text-embedding-004` model.
  - User queries are embedded, and relevant chunks are retrieved based on cosine similarity.
  - Retrieved chunks, chat history, and the user query are used to construct a prompt for a Gemini model (e.g., `gemini-1.5-flash-latest`).
- **Persistent Conversations:** Chats and messages are stored in an SQLite database, allowing users to continue conversations.
- **Multiple Chats per User:** Users (identified by an `X-User-ID` header) can have multiple independent chat sessions.
- **Decline to Answer:** The Gemini model is instructed via a system prompt to state if it doesn't have the information based on the provided context.
- **Negative Feedback:** Users can provide negative feedback on model messages.
- **Auto-generated Chat Titles:** New chats get an automatically generated title based on the initial user query, using Gemini.
- **Dockerfile:** For containerizing the application.
- **Makefile:** For common development tasks (build, run, ingest data, clean).

## Prerequisites

- Go (version 1.23 or later)
- A Google Gemini API Key (from Google AI Studio)
- `make`
- `docker`

## Project Structure

```bash
gwi-jedi-team-challenge/
├── cmd/server/main.go              # Main application
├── internal/
│   ├── api/                        # HTTP handlers and routing
│   ├── config/                     # Configuration loading
│   ├── core/                       # Core business logic (RAG, chat, LLM)
│   ├── store/                      # Database interaction (SQLite)
│   └── utils/                      # Utility functions (embedding math)
├── data.md                         # Provided data (Markdown table format)
├── Dockerfile
├── Makefile
├── go.mod
├── go.sum
├── .env.example                    # Example environment variables
└── README.md
```

## Setup and Running

1. **Clone the Repository:**

    ```bash
    git clone git@github.com:kiraleos/jedi-team-challenge.git
    cd jedi-team-challenge
    ```

2. **Configuration:**
    Copy the example environment file and fill in your Gemini API Key:

    ```bash
    cp .env.example .env
    ```

    Edit `.env` and set your `GEMINI_API_KEY`:

    ```bash
    GEMINI_API_KEY="your-google-ai-studio-gemini-api-key"
    DATABASE_URL="gwi_chatbot.db"
    HTTP_PORT="8080"
    LOG_LEVEL="INFO"
    ```

3. **Install Dependencies:**

    ```bash
    go mod tidy
    ```

4. **Data Ingestion (Important First Step):**
    The market research data from `data.md` needs to be processed (each row parsed, embedded using Gemini) and stored in the database. Run the following command:

    ```bash
    make ingest_data
    ```

    This command will read `data.md`, connect to Google's Gemini API to generate embeddings for each data row, and store them in the `gwi_chatbot.db` SQLite file. **This might take up to 20 minutes and for this reason you can trim the data in `data.md` to a smaller size for testing purposes.**

5. **Run the Application:**
    Using Makefile:

    ```bash
    make run
    ```

    Alternatively, build and run manually:

    ```bash
    make build
    ./jedi-team-challenge
    ```

    The server will start, typically on port `8080` (configurable via `HTTP_PORT` in `.env`).

## Running with Docker

1. **Ensure `.env` is configured** as described above (Gemini API key is needed).
2. **Ingest data first** on your host machine so `gwi_chatbot.db` is populated:

    ```bash
    make ingest_data
    ```

3. **Build the Docker Image:**

    ```bash
    docker build -t jedi-team-challenge .
    ```

4. **Run the Docker Container:**
    You need to pass the Gemini API key as an environment variable and mount the database file for persistence.

    ```bash
    # Ensure gwi_chatbot.db exists (run `make ingest_data` on host first)
    docker run -p 8080:8080 \
      -e GEMINI_API_KEY="your_gemini_api_key_from_env_or_direct" \
      -v $(pwd)/gwi_chatbot.db:/root/gwi_chatbot.db \
      jedi-team-challenge
    ```

    Note: The `DATABASE_URL` inside the container will default to `gwi_chatbot.db` (relative to workdir `/root/`).

## API Endpoints

All endpoints require an `X-User-ID` header (e.g., `X-User-ID: user123`).

- **`POST /api/chats`**: Create a new chat.
  - Request Body (optional): `{ "first_message": "Initial query for the chat" }`
  - Response: `201 Created` with chat details and initial messages (if any).

    ```json
    {
        "id": "chat-uuid-...",
        "user_id": 1, // internal DB user ID
        "title": null, // null initially, will be generated asynchronously
        "created_at": "...",
        "messages": [ /* user message and model response if first_message was provided */ ]
    }
    ```

- **`GET /api/chats`**: List all chats for the authenticated user.
  - Response: `200 OK` with an array of chat objects.

    ```json
    [
        { "id": "chat-uuid-1", "user_id": 1, "title": "...", "created_at": "..." }
    ]
    ```

- **`GET /api/chats/{chatID}`**: Get details and messages for a specific chat.
  - Response: `200 OK` with chat details and an array of messages.

    ```json
    {
        "id": "chat-uuid-1",
        "user_id": 1,
        "title": "...",
        "created_at": "...",
        "messages": [
            { "id": "msg-uuid-1", "chat_id": "chat-uuid-1", "sender": "user", "content": "...", "timestamp": "...", "negative_feedback": false }
        ]
    }
    ```

- **`POST /api/chats/{chatID}/messages`**: Send a new message to a chat.
  - Request Body: `{ "content": "User's question" }`
  - Response: `200 OK` with the model's reply message object.

    ```json
    {
        "id": "msg-uuid-model-reply",
        "chat_id": "chat-uuid-1",
        "sender": "model",
        "content": "model's answer...",
        "timestamp": "...",
        "negative_feedback": false
    }
    ```

- **`POST /api/messages/{messageID}/feedback`**: Mark a message with negative feedback.
  - Request Body: `{ "negative": true }` (or `false` to undo)
  - Response: `204 No Content` on success.

- **`GET /api/health`**: Health check endpoint (public, no auth needed).
  - Response: `200 OK` with `{"status":"ok"}`

### Example `curl` commands

Assume `X-User-ID: 1` and server running on `localhost:8080`.

1. **Create a new chat with an initial question:**

    ```bash
    curl -X POST -H "Content-Type: application/json" -H "X-User-ID: 1" \
      -d '{ "first_message": "How likely are Gen Z in Nashville to find brands via vlogs?" }' \
      http://localhost:8080/api/chats
    ```

    (Note the `id` of the chat from the response, let's say it's `abc-123`)

2. **Send a follow-up message to that chat:**

    ```bash
    curl -X POST -H "Content-Type: application/json" -H "X-User-ID: 1" \
      -d '{ "content": "What about anxiety from social media for them?" }' \
      http://localhost:8080/api/chats/abc-123/messages
    ```

    (Note the `id` of the model's message from the response, let's say it's `msg-xyz-789`)

3. **Provide negative feedback on the model's last message:**

    ```bash
    curl -X POST -H "Content-Type: application/json" -H "X-User-ID: 1" \
      -d '{ "negative": true }' \
      http://localhost:8080/api/messages/msg-xyz-789/feedback
    ```

## Design Choices & Assumptions

- **User Identification:** Users are identified by a string passed in the `X-User-ID` header. The application auto-creates an internal user record if one doesn't exist.
- **Data Format & Chunking:** `data.md` is expected to be a Markdown table where each row's `text` column represents a single fact or chunk. The ingestion process parses these rows.
- **Embeddings & LLM:** Google's Gemini models are used (e.g., `text-embedding-004` for embeddings, `gemini-1.5-flash-latest` for generation). Embeddings are stored as JSON strings in SQLite.
- **Context for LLM:** The Gemini prompt includes:
    1. A system instruction defining its role and behavior.
    2. Recent chat history for conversational context.
    3. Top K (currently 3) relevant data rows retrieved from `data.md` via similarity search.
    4. The current user query.
- **Title Generation:** Chat titles are generated asynchronously after the first user message using Gemini.
- **SQLite:** Chosen as per requirements for persistence. For higher concurrency or larger datasets, a more robust database with vector capabilities would be considered.
- **In-memory Chunk Cache:** The RAG service loads all data chunks and their embeddings into memory at startup for faster similarity searches. This is feasible for the expected size of `data.md`.

## Potential Future Enhancements

- **Vector Database Integration:** Replace SQLite for embeddings with a dedicated vector database for scalability.
- **Transactional Support:** Implement transactions for chat and message creation to ensure atomicity.
- **Advanced User Authentication:** Implement OAuth2 or JWT-based authentication.
- **Testing:** Add unit and integration tests for all components. (Not included due to time constraints)
