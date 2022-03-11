CREATE TABLE IF NOT EXISTS votes (
	-- take the uuid from microsoft by calling an API endpoint with provided accesstoken
	student VARCHAR(256) NOT NULL,

	-- 0 if downvote, 1 if upvote
	vote_type int NOT NULL,
	song_id int NOT NULL,

	submitted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
