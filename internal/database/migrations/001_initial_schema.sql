-- Create UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create platforms table
CREATE TABLE IF NOT EXISTS platforms (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    api_endpoint VARCHAR(500) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create programs table
CREATE TABLE IF NOT EXISTS programs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(500) NOT NULL,
    platform VARCHAR(255) NOT NULL,
    url VARCHAR(500) NOT NULL,
    program_url VARCHAR(500) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_updated TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(platform, url)
);

-- Create assets table
CREATE TABLE IF NOT EXISTS assets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    program_id UUID NOT NULL REFERENCES programs(id) ON DELETE CASCADE,
    url VARCHAR(500) NOT NULL,
    domain VARCHAR(255) NOT NULL,
    subdomain VARCHAR(255),
    ip VARCHAR(45), -- IPv6 compatible
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    source VARCHAR(50) NOT NULL DEFAULT 'chaosdb',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(program_id, url)
);

-- Create asset_responses table
CREATE TABLE IF NOT EXISTS asset_responses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    status_code INTEGER NOT NULL,
    headers TEXT, -- JSON encoded headers
    body TEXT,
    response_time BIGINT NOT NULL, -- in milliseconds
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create scans table
CREATE TABLE IF NOT EXISTS scans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    program_id UUID NOT NULL REFERENCES programs(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'running',
    assets_found INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for better performance (only if they don't exist)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_programs_platform') THEN
        CREATE INDEX idx_programs_platform ON programs(platform);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_programs_is_active') THEN
        CREATE INDEX idx_programs_is_active ON programs(is_active);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_programs_last_updated') THEN
        CREATE INDEX idx_programs_last_updated ON programs(last_updated);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_assets_program_id') THEN
        CREATE INDEX idx_assets_program_id ON assets(program_id);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_assets_domain') THEN
        CREATE INDEX idx_assets_domain ON assets(domain);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_assets_status') THEN
        CREATE INDEX idx_assets_status ON assets(status);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_assets_source') THEN
        CREATE INDEX idx_assets_source ON assets(source);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_assets_created_at') THEN
        CREATE INDEX idx_assets_created_at ON assets(created_at);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_asset_responses_asset_id') THEN
        CREATE INDEX idx_asset_responses_asset_id ON asset_responses(asset_id);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_asset_responses_status_code') THEN
        CREATE INDEX idx_asset_responses_status_code ON asset_responses(status_code);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_asset_responses_created_at') THEN
        CREATE INDEX idx_asset_responses_created_at ON asset_responses(created_at);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_scans_program_id') THEN
        CREATE INDEX idx_scans_program_id ON scans(program_id);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_scans_status') THEN
        CREATE INDEX idx_scans_status ON scans(status);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_scans_started_at') THEN
        CREATE INDEX idx_scans_started_at ON scans(started_at);
    END IF;
END $$;

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at (only if they don't exist)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_programs_updated_at') THEN
        CREATE TRIGGER update_programs_updated_at BEFORE UPDATE ON programs
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_assets_updated_at') THEN
        CREATE TRIGGER update_assets_updated_at BEFORE UPDATE ON assets
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_scans_updated_at') THEN
        CREATE TRIGGER update_scans_updated_at BEFORE UPDATE ON scans
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;

-- Insert default platforms
INSERT INTO platforms (name, api_endpoint) VALUES
    ('hackerone', 'https://api.hackerone.com/v1'),
    ('bugcrowd', 'https://api.bugcrowd.com')
ON CONFLICT (name) DO NOTHING;
