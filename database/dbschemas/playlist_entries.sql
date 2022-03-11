CREATE TABLE IF NOT EXISTS playlist_entries (

	playlist_id int NOT NULL,
	song_id int NOT NULL,

	debuted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
