CREATE TABLE IF NOT EXISTS telemetry_events (
    event_id BIGSERIAL PRIMARY KEY,
    prosthesis_id TEXT NOT NULL,
    event_time TIMESTAMPTZ NOT NULL,
    movement_type TEXT NOT NULL,
    response_time_ms INTEGER NOT NULL,
    battery_level INTEGER NOT NULL,
    signal_quality NUMERIC(5, 2) NOT NULL
);

INSERT INTO telemetry_events (
    prosthesis_id,
    event_time,
    movement_type,
    response_time_ms,
    battery_level,
    signal_quality
) VALUES
    ('bp-arm-001', now() - interval '6 hours', 'grip', 84, 88, 0.94),
    ('bp-arm-001', now() - interval '5 hours', 'release', 91, 86, 0.91),
    ('bp-arm-001', now() - interval '4 hours', 'rotate', 103, 82, 0.88),
    ('bp-arm-002', now() - interval '5 hours', 'grip', 75, 93, 0.96),
    ('bp-arm-002', now() - interval '3 hours', 'release', 79, 91, 0.95),
    ('bp-hand-101', now() - interval '8 hours', 'pinch', 97, 77, 0.90),
    ('bp-hand-101', now() - interval '7 hours', 'grip', 109, 74, 0.84),
    ('bp-hand-101', now() - interval '2 hours', 'release', 98, 69, 0.89),
    ('bp-hand-102', now() - interval '4 hours', 'grip', 88, 80, 0.93),
    ('bp-hand-102', now() - interval '1 hours', 'rotate', 92, 76, 0.92),
    ('bp-hand-103', now() - interval '3 hours', 'pinch', 113, 65, 0.81),
    ('bp-hand-103', now() - interval '1 hours', 'release', 101, 61, 0.86);
