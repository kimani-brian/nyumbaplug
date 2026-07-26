ALTER TABLE properties ADD COLUMN county text;
ALTER TABLE properties ADD COLUMN maps_url text;
ALTER TABLE properties ADD COLUMN image_url text;

CREATE TABLE unit_categories (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id uuid NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    name text NOT NULL,
    description text,
    rent_amount numeric NOT NULL,
    quantity_available integer NOT NULL DEFAULT 0,
    photos jsonb DEFAULT '[]'::jsonb,
    video_url text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_unit_categories_property_id ON unit_categories(property_id);
