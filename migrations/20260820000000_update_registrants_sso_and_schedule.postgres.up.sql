-- Hapus aturan NOT NULL agar bisa mendaftar hanya dengan Email & Nama
ALTER TABLE registrants ALTER COLUMN angkatan DROP NOT NULL;
ALTER TABLE registrants ALTER COLUMN prodi DROP NOT NULL;
ALTER TABLE registrants ALTER COLUMN fakultas DROP NOT NULL;
ALTER TABLE registrants ALTER COLUMN domicile DROP NOT NULL;
ALTER TABLE registrants ALTER COLUMN phone DROP NOT NULL;
ALTER TABLE registrants ALTER COLUMN password DROP NOT NULL;
ALTER TABLE registrants ALTER COLUMN division_1 DROP NOT NULL;
ALTER TABLE registrants ALTER COLUMN division_2 DROP NOT NULL;
ALTER TABLE registrants ALTER COLUMN swot_s DROP NOT NULL;
ALTER TABLE registrants ALTER COLUMN swot_w DROP NOT NULL;
ALTER TABLE registrants ALTER COLUMN swot_o DROP NOT NULL;
ALTER TABLE registrants ALTER COLUMN swot_t DROP NOT NULL;
ALTER TABLE registrants ALTER COLUMN organization_exp DROP NOT NULL;
ALTER TABLE registrants ALTER COLUMN commitment DROP NOT NULL;
ALTER TABLE registrants ALTER COLUMN cv_url DROP NOT NULL;

-- Tambahkan kolom untuk jadwal screening
ALTER TABLE registrants ADD COLUMN screening_date TIMESTAMP;
ALTER TABLE registrants ADD COLUMN screening_location TEXT;
ALTER TABLE registrants ADD COLUMN screening_link TEXT;