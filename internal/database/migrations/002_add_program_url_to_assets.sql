-- Migration: Add program_url to assets table
-- This allows tracking which program each asset belongs to directly

-- Add program_url column to assets table
ALTER TABLE assets ADD COLUMN IF NOT EXISTS program_url VARCHAR(500);

-- Update existing assets to have program_url from their associated program
UPDATE assets 
SET program_url = programs.program_url 
FROM programs 
WHERE assets.program_id = programs.id;

-- Make program_url NOT NULL after populating existing data
ALTER TABLE assets ALTER COLUMN program_url SET NOT NULL;

-- Add index for program_url for better query performance
CREATE INDEX IF NOT EXISTS idx_assets_program_url ON assets(program_url);

-- Add comment to document the new field
COMMENT ON COLUMN assets.program_url IS 'The URL of the program this asset belongs to, for direct reference without joining';
