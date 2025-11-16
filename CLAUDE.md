# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

WUBRG Voting Bot is a Telegram bot written in Go that manages user interactions through a stateful dialog system and poll creation functionality. The bot is designed to handle multi-step conversations with users, maintaining session state across messages.

## Technology Stack

- **Language**: Go 1.24
- **Telegram Bot Framework**: gopkg.in/telebot.v4 (v4.0.0-beta.7)
- **Database**: PostgreSQL via github.com/jackc/pgx/v5 (v5.7.6)
- **Architecture**: Stateful dialog-based interaction model

## Development Commands

### Running the Bot
```bash
go run main.go
```

### Building
```bash
go build -o wubrg-voting-bot
```

### Testing
```bash
go test ./...
```

### Running Tests for a Specific Package
```bash
go test ./bot/...
```

## Environment Variables

The bot requires the following environment variables:

- `BOT_TOKEN` (required): Telegram Bot API token
- `DATABASE_URL` (optional): PostgreSQL connection string. Defaults to `postgresql://postgres:postgres@localhost:5432/wubrg_voting`

## Architecture

### Core Components

1. **main.go**: Application entry point
   - Initializes environment variables
   - Creates database connection to PostgreSQL
   - Creates bot instance with database connection
   - Verifies database connectivity on startup

2. **bot/bot.go**: Main bot logic
   - Bot struct holds telebot instance, database connection, and DialogManager
   - Command handler registration
   - Routing logic based on dialog state
   - Available commands: `/start`, `/help`, `/status`, `/cancel`, `/createpoll`, `/listpolls`, `/publishpoll`

3. **bot/dialog.go**: Stateful dialog management system
   - `DialogManager`: Thread-safe session manager using sync.RWMutex
   - `DialogContext`: Per-user state and data storage
   - State machine with states: `idle`, `create_poll_title`, `create_poll_option`, `create_poll_confirm`
   - In-memory session storage (data persists only while bot is running)

4. **bot/poll.go**: Poll creation workflow
   - Multi-step poll creation dialog
   - Validates poll title (3-200 characters) and options (1-100 characters, minimum 2 options)
   - Supports Russian and English confirmation keywords
   - `savePollToDB()`: Saves poll data to PostgreSQL in a transaction
     * Inserts poll into `voting.polls` table
     * Inserts options into `voting.poll_options` table
     * Returns generated poll ID on success
     * Rolls back transaction on any error

### Key Architectural Patterns

**State Machine Dialog Flow**: The bot uses a state machine pattern where each user has a `DialogContext` that tracks their current state and associated data. Text message handlers are routed based on the user's current state, enabling multi-turn conversations.

**Thread-Safe Session Management**: The `DialogManager` uses read-write locks to safely manage concurrent user sessions. Each user's state is isolated in their own `DialogContext`.

**Handler Routing**: The `handleText` function in bot.go acts as a router, dispatching incoming text messages to appropriate handlers based on the user's current dialog state.

**Data Persistence Model**: 
- Dialog session data is stored in-memory (temporary, for conversation flow)
- Poll data is persisted to PostgreSQL database:
  * `handlePollConfirmInput` calls `savePollToDB()` to save poll and options
  * Uses transactions to ensure data consistency
  * Returns poll ID to user after successful creation

## Common Patterns

### Adding a New Command

1. Add handler method to Bot struct in `bot/bot.go`
2. Register handler in `registerHandlers()` method
3. Update help text in `handleHelp()`

### Adding a New Dialog Flow

1. Define new states in `bot/dialog.go` State constants
2. Add state handler in `handleText()` switch statement in `bot/bot.go`
3. Implement handler methods following the pattern of existing handlers
4. Use `DialogManager` methods: `SetState()`, `SetData()`, `GetData()`, `ResetContext()`

### Dialog State Transitions

All dialog flows follow this pattern:
1. Command handler resets context and sets initial state
2. User input handlers validate input and store data
3. Progress through states using `SetState()`
4. Final handler saves data (currently logs) and returns to `StateIdle`
5. Use `/cancel` command to abort any dialog

## Database Schema

The bot uses PostgreSQL with a custom schema `voting`. Key tables:

- `voting.polls`: Poll metadata (title, creator, status, timestamps)
- `voting.poll_options`: Poll options/choices
- `voting.poll_chats`: Tracks where polls are published
- `voting.votes`: User votes on polls

See `db-schema/` directory for:
- `schema.sql`: Full database schema with indexes
- `queries.sql`: Useful queries for data analysis
- `sample_data.sql`: Test data
- `drop_tables.sql`: Schema cleanup
- `README.md`: Detailed schema documentation

## Important Notes

- PostgreSQL database must be running and accessible for bot to start
- Dialog data is stored in-memory and will be lost on bot restart
- Poll data is persisted to database and survives restarts
- The bot supports both Russian and English keywords for confirmation
- Thread-safety is critical when modifying DialogManager methods
- User validation happens inline with helpful error messages to guide users
- All database operations use transactions for data consistency

## Testing

See `TESTING.md` for detailed testing instructions including:
- Environment setup
- Database initialization
- Step-by-step poll creation testing
- Validation testing
- Troubleshooting

## Integration Status

✅ **Implemented:**
- Database connection and verification
- Poll creation with database persistence
- Transaction-based data saving
- Error handling for database operations
- Poll publishing to chats
- Voting functionality (inline buttons)
- Vote recording to database
- Poll results display
- Inline mode for poll sharing
