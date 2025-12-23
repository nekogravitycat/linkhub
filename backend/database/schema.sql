-- =============================================
-- Extensions
-- =============================================

-- Enable pg_trgm for efficient fuzzy searching (LIKE '%keyword%')
-- This is critical for the performance of searching slugs or URLs.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =============================================
-- Tables
-- =============================================

CREATE TABLE IF NOT EXISTS links (
    id BIGSERIAL PRIMARY KEY,
    -- Slug must be unique. Using TEXT is preferred over VARCHAR in Postgres
    -- as there is no performance penalty and it offers flexibility.
    slug TEXT NOT NULL UNIQUE,
    url TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- =============================================
-- Automation Logic (Triggers)
-- =============================================

-- Function to automatically update the 'updated_at' timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to execute the function before any update on the 'links' table
DROP TRIGGER IF EXISTS update_links_updated_at ON links;
CREATE TRIGGER update_links_updated_at
    BEFORE UPDATE ON links
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =============================================
-- Indexes (Performance Optimization)
-- =============================================

-- Sorting Optimization
-- Optimized for your default sort: Created At (Newest -> Oldest)
CREATE INDEX IF NOT EXISTS idx_links_created_at_desc ON links(created_at DESC);

-- Fuzzy Search Optimization
-- These GIN indexes allow high-performance 'ILIKE %keyword%' queries.
-- Without these, searching 100k+ rows will result in slow full-table scans.
CREATE INDEX IF NOT EXISTS idx_links_slug_trgm ON links USING gin (slug gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_links_url_trgm ON links USING gin (url gin_trgm_ops);
