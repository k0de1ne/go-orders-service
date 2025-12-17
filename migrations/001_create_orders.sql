CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY,
    product VARCHAR(255) NOT NULL,
    quantity INT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
