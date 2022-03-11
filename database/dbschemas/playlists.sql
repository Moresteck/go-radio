CREATE TABLE IF NOT EXISTS playlists (
	playlist_id int PRIMARY KEY NOT NULL AUTO_INCREMENT,

	title VARCHAR(256) NOT NULL,
	description VARCHAR(256) NOT NULL,
	-- 0 is false, 1 is true
	ranking int NOT NULL,

	debuted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
