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

CREATE TABLE IF NOT EXISTS product_queue (
    -- It is best practice to have an internal integer ID
                                             id SERIAL PRIMARY KEY,

    -- 'url' must still be UNIQUE to satisfy your query's conflict clause
                                             url TEXT NOT NULL UNIQUE,

                                             domain TEXT NOT NULL,

    -- Setting a default allows you to omit it in other queries if needed
                                             status VARCHAR(50) NOT NULL DEFAULT 'pending',

    -- Automatically track when the row was created
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
                             );

-- Optional: Create an index on status if you plan to query by "pending" often
CREATE INDEX IF NOT EXISTS idx_product_queue_status ON product_queue(status);