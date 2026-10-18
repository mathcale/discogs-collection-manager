UPDATE users SET email = lower(trim(email));

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower ON users (lower(email));
