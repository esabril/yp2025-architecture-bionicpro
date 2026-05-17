CREATE TABLE IF NOT EXISTS customers (
    user_id TEXT PRIMARY KEY,
    keycloak_username TEXT NOT NULL UNIQUE,
    full_name TEXT NOT NULL,
    email TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS prostheses (
    prosthesis_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES customers(user_id),
    model TEXT NOT NULL,
    serial_number TEXT NOT NULL,
    issued_at DATE NOT NULL
);

INSERT INTO customers (user_id, keycloak_username, full_name, email) VALUES
    ('cust-001', 'user1', 'User One', 'user1@example.com'),
    ('cust-002', 'user2', 'User Two', 'user2@example.com'),
    ('cust-101', 'prothetic1', 'Prothetic One', 'prothetic1@example.com'),
    ('cust-102', 'prothetic2', 'Prothetic Two', 'prothetic2@example.com'),
    ('cust-103', 'prothetic3', 'Prothetic Three', 'prothetic3@example.com')
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO prostheses (prosthesis_id, user_id, model, serial_number, issued_at) VALUES
    ('bp-arm-001', 'cust-001', 'BionicPRO Arm S', 'SN-BP-001', '2025-11-12'),
    ('bp-arm-002', 'cust-002', 'BionicPRO Arm S', 'SN-BP-002', '2025-12-04'),
    ('bp-hand-101', 'cust-101', 'BionicPRO Hand M', 'SN-BP-101', '2026-01-18'),
    ('bp-hand-102', 'cust-102', 'BionicPRO Hand M', 'SN-BP-102', '2026-02-07'),
    ('bp-hand-103', 'cust-103', 'BionicPRO Hand L', 'SN-BP-103', '2026-02-19')
ON CONFLICT (prosthesis_id) DO NOTHING;
