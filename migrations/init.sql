CREATE TABLE IF NOT EXISTS pages (
                                 id SERIAL PRIMARY KEY,
                                 url TEXT UNIQUE NOT NULL,
                                 title TEXT,
                                 content_text TEXT,
                                 status_code INT,
                                 load_time_ms INT,
                                 crawled_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS page_links (
                                      source_url TEXT NOT NULL,
                                      target_url TEXT NOT NULL,
                                      PRIMARY KEY (source_url, target_url)
);