-- Add free-text details (scam message) to property reports
ALTER TABLE property_reports ADD COLUMN details TEXT NOT NULL DEFAULT '';
