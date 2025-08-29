CREATE TABLE posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		content TEXT,
		published BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);
CREATE TABLE sqlite_sequence(name,seq);
CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
INSERT INTO users (id, username, email, created_at) VALUES (1, 'alice', 'alice@example.com', '2025-08-29 17:31:49');
INSERT INTO users (id, username, email, created_at) VALUES (2, 'bob', 'bob@example.com', '2025-08-29 17:31:49');
INSERT INTO posts (id, user_id, title, content, published, created_at) VALUES (1, 1, 'Hello World! (Updated)', 'This post has been updated to show that UPDATE works!', 1, '2025-08-29 17:31:49');
