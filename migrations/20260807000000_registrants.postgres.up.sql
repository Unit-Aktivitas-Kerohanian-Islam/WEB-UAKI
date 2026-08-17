CREATE TYPE division_choice AS ENUM (
    'KP', 'MENTORING', 'SYIAR', 'CM', 'HUMAS', 'MUCC', 'RAB', 'EKRAF'
);

CREATE TABLE registrants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    nim TEXT NOT NULL UNIQUE,
    angkatan TEXT NOT NULL,
    prodi TEXT NOT NULL,
    fakultas TEXT NOT NULL,
    domicile TEXT NOT NULL,
    phone TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    division_1 division_choice NOT NULL,
    division_2 division_choice NOT NULL,
    swot_s TEXT NOT NULL,
    swot_w TEXT NOT NULL,
    swot_o TEXT NOT NULL,
    swot_t TEXT NOT NULL,
    organization_exp TEXT NOT NULL,
    commitment TEXT NOT NULL,
    cv_url TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);