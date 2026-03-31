CREATE TABLE IF NOT EXISTS sync_processed_events (
    event_id UUID PRIMARY KEY,
    source_node_id VARCHAR(64),
    table_name VARCHAR(64),
    processed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS clientes (
    id INTEGER PRIMARY KEY,
    nome VARCHAR(100),
    data_nasc TIMESTAMP
);