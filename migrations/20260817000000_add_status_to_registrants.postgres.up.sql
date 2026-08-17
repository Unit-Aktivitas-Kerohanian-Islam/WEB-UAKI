CREATE TYPE registrant_status AS ENUM ('PENDING', 'LOLOS_BERKAS', 'DITOLAK');

ALTER TABLE registrants 
ADD COLUMN status registrant_status NOT NULL DEFAULT 'PENDING';