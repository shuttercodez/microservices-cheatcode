-- Users table for OAuth 2.0 Password Grant authentication
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(64) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- OAuth 2.0 Clients table for Client Credentials Grant (M2M authentication)
CREATE TABLE IF NOT EXISTS oauth_clients (
    client_id VARCHAR(64) PRIMARY KEY,
    client_secret_hash VARCHAR(255) NOT NULL,
    name VARCHAR(128) NOT NULL,
    scope VARCHAR(255) NOT NULL DEFAULT 'inventory:read inventory:write orders:read orders:write',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default admin user (username: "admin", password: "adminpassword") using pgcrypto Blowfish/Bcrypt
INSERT INTO users (username, password_hash, role)
VALUES ('admin', crypt('adminpassword', gen_salt('bf', 10)), 'admin')
ON CONFLICT (username) DO NOTHING;

-- Seed default OAuth 2.0 M2M client (client_id: "inventory-client", client_secret: "inventory-secret")
INSERT INTO oauth_clients (client_id, client_secret_hash, name, scope)
VALUES (
    'inventory-client',
    crypt('inventory-secret', gen_salt('bf', 10)),
    'Default Inventory M2M Client',
    'inventory:read inventory:write orders:read orders:write'
)
ON CONFLICT (client_id) DO NOTHING;
