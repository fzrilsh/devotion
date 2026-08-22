CREATE TABLE province (
    code text PRIMARY KEY,
    name text NOT NULL,
    CONSTRAINT province_code_format CHECK (code ~ '^[0-9]{2}$')
);

CREATE TABLE city (
    code          text PRIMARY KEY,
    province_code text NOT NULL REFERENCES province(code) ON DELETE RESTRICT,
    name          text NOT NULL,
    CONSTRAINT city_code_format CHECK (code ~ '^[0-9]{4}$'),
    CONSTRAINT city_belongs_to_province CHECK (left(code, 2) = province_code)
);

CREATE INDEX idx_city_province ON city (province_code);
