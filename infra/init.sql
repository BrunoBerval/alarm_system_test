CREATE TABLE IF NOT EXISTS alarms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'OPEN',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP WITH TIME ZONE
);

-- Deduplicação atômica.
-- Regra:"No máximo um alarme OPEN por dispositivo"
CREATE UNIQUE INDEX IF NOT EXISTS uniq_open_alarm_per_device
    ON alarms (device_id)
    WHERE status = 'OPEN';

-- Suporta o ORDER BY created_at DESC do GET /alarms.
CREATE INDEX IF NOT EXISTS idx_alarms_created_at
    ON alarms (created_at DESC);
