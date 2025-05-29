package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	database := &Database{db: db}
	if err := database.createTables(); err != nil {
		return nil, err
	}

	return database, nil
}

func (d *Database) createTables() error {
	createTablesSQL := `
	CREATE TABLE IF NOT EXISTS channels (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS messages (
		ts TEXT PRIMARY KEY,
		channel_id TEXT NOT NULL,
		user_id TEXT,
		text TEXT,
		thread_ts TEXT,
		reply_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (channel_id) REFERENCES channels(id)
	);

	CREATE TABLE IF NOT EXISTS replies (
		ts TEXT PRIMARY KEY,
		thread_ts TEXT NOT NULL,
		channel_id TEXT NOT NULL,
		user_id TEXT,
		text TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (thread_ts) REFERENCES messages(ts),
		FOREIGN KEY (channel_id) REFERENCES channels(id)
	);

	CREATE INDEX IF NOT EXISTS idx_messages_channel_id ON messages(channel_id);
	CREATE INDEX IF NOT EXISTS idx_messages_thread_ts ON messages(thread_ts);
	CREATE INDEX IF NOT EXISTS idx_replies_thread_ts ON replies(thread_ts);
	CREATE INDEX IF NOT EXISTS idx_replies_channel_id ON replies(channel_id);
	`

	_, err := d.db.Exec(createTablesSQL)
	return err
}

func (d *Database) SaveChannel(id, name string) error {
	_, err := d.db.Exec("INSERT OR REPLACE INTO channels (id, name) VALUES (?, ?)", id, name)
	return err
}

func (d *Database) SaveMessage(ts, channelID, userID, text, threadTS string, replyCount int) error {
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO messages (ts, channel_id, user_id, text, thread_ts, reply_count) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		ts, channelID, userID, text, threadTS, replyCount)
	return err
}

func (d *Database) SaveReply(ts, threadTS, channelID, userID, text string) error {
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO replies (ts, thread_ts, channel_id, user_id, text) 
		VALUES (?, ?, ?, ?, ?)`,
		ts, threadTS, channelID, userID, text)
	return err
}

func (d *Database) GetLastMessageTimestamp(channelID string) (string, error) {
	var ts string
	err := d.db.QueryRow("SELECT ts FROM messages WHERE channel_id = ? ORDER BY ts DESC LIMIT 1", channelID).Scan(&ts)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return ts, err
}

type MessageWithReplies struct {
	Timestamp    string
	ChannelID    string
	ChannelName  string
	UserID       string
	Text         string
	ThreadTS     string
	ReplyCount   int
	Replies      []Reply
}

type Reply struct {
	Timestamp string
	UserID    string
	Text      string
}

func (d *Database) GetAllMessagesWithReplies(channelID string) ([]MessageWithReplies, error) {
	query := `
		SELECT m.ts, m.channel_id, c.name, m.user_id, m.text, m.thread_ts, m.reply_count
		FROM messages m
		JOIN channels c ON m.channel_id = c.id
		WHERE m.channel_id = ?
		ORDER BY m.ts ASC`

	rows, err := d.db.Query(query, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []MessageWithReplies
	for rows.Next() {
		var msg MessageWithReplies
		err := rows.Scan(&msg.Timestamp, &msg.ChannelID, &msg.ChannelName, &msg.UserID, &msg.Text, &msg.ThreadTS, &msg.ReplyCount)
		if err != nil {
			return nil, err
		}

		if msg.ThreadTS != "" && msg.ReplyCount > 0 {
			replies, err := d.getReplies(msg.ThreadTS)
			if err != nil {
				return nil, err
			}
			msg.Replies = replies
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func (d *Database) getReplies(threadTS string) ([]Reply, error) {
	query := `
		SELECT ts, user_id, text
		FROM replies
		WHERE thread_ts = ?
		ORDER BY ts ASC`

	rows, err := d.db.Query(query, threadTS)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var replies []Reply
	for rows.Next() {
		var reply Reply
		err := rows.Scan(&reply.Timestamp, &reply.UserID, &reply.Text)
		if err != nil {
			return nil, err
		}
		replies = append(replies, reply)
	}

	return replies, nil
}

func (d *Database) GetChannels() (map[string]string, error) {
	query := "SELECT id, name FROM channels"
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	channels := make(map[string]string)
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		channels[id] = name
	}

	return channels, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}