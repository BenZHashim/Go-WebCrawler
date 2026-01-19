CREATE TABLE crawled_pages (
                               id SERIAL PRIMARY KEY,
                               url TEXT UNIQUE NOT NULL,
                               title TEXT,
                               h1 TEXT,
                               crawled_at TIMESTAMP
);