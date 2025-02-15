\c battleship;
/*
To run this script delete the image and volume of the database container
*/



/*
This table stores user data with:
- id: Unique identifier
- username: Unique player name
- rating: Elo-style rating (default 1000)
*/
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    rating INT DEFAULT 1000,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


