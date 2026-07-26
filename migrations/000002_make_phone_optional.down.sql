ALTER TABLE users ALTER COLUMN phone SET NOT NULL;

ALTER TABLE units ADD CONSTRAINT units_unit_type_check CHECK (unit_type IN ('studio', '1br', '2br', '3br', 'other'));
