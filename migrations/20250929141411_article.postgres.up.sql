CREATE TYPE article_category AS ENUM (
    'islam',
    'umum',
    'informasi'
);

CREATE TABLE articles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id UUID REFERENCES admins(id) ON DELETE SET NULL,
    category article_category NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    title TEXT NOT NULL,
    value TEXT NOT NULL,
    img_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);