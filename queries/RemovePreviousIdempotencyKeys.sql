DELETE FROM idempotency_keys WHERE processed_at < datetime('now', '-30 days');
