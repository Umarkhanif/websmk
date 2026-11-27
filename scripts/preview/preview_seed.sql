-- Preview sekolah fiksi "SMK Cendekia Nusantara" untuk instance sales/demo.
-- Hanya dijalankan di database preview terpisah (lihat scripts/preview.sh).
-- Idempotent: aman dijalankan ulang.
-- Semua data di bawah ini fiktif; tidak ada orang/siswa/perusahaan nyata.

INSERT INTO vocational_majors (code, name) VALUES
    ('TKJ',  'Teknik Komputer dan Jaringan'),
    ('RPL',  'Rekayasa Perangkat Lunak'),
    ('TBSM', 'Teknik dan Bisnis Sepeda Motor'),
    ('AKL',  'Akuntansi dan Keuangan Lembaga')
ON CONFLICT (code) DO NOTHING;

INSERT INTO classrooms (major_code, grade_level, name) VALUES
    ('TKJ', 10, 'X TKJ 1'),   ('TKJ', 11, 'XI TKJ 1'),  ('TKJ', 12, 'XII TKJ 1'),
    ('RPL', 10, 'X RPL 1'),   ('RPL', 11, 'XI RPL 1'),  ('RPL', 12, 'XII RPL 1'),
    ('TBSM', 10, 'X TBSM 1'), ('TBSM', 11, 'XI TBSM 1'),('TBSM', 12, 'XII TBSM 1'),
    ('AKL', 10, 'X AKL 1'),   ('AKL', 11, 'XI AKL 1'),  ('AKL', 12, 'XII AKL 1')
ON CONFLICT DO NOTHING;

-- Siswa fiktif untuk angka "Siswa aktif" di landing (24 per rombel).
-- Akun siswa preview sengaja non-loginable (password_hash '!'); hanya untuk hitungan.
INSERT INTO users (id, identifier, full_name, role, password_hash)
SELECT 'preview-stu-' || c.id::text || '-' || g.n,
       'preview-stu-' || c.id::text || '-' || g.n,
       'Siswa Demo ' || g.n,
       'student', '!'
FROM classrooms c CROSS JOIN generate_series(1, 24) AS g(n)
ON CONFLICT (id) DO NOTHING;

INSERT INTO students (user_id, classroom_id, full_name)
SELECT 'preview-stu-' || c.id::text || '-' || g.n, c.id, 'Siswa Demo ' || g.n
FROM classrooms c CROSS JOIN generate_series(1, 24) AS g(n)
WHERE NOT EXISTS (
    SELECT 1 FROM students s
    WHERE s.user_id = 'preview-stu-' || c.id::text || '-' || g.n
);

-- Data master tambahan yang dibaca halaman publik (renderHomePage).
-- Merge, jangan replace: kunci lain di app_state tetap utuh.
INSERT INTO app_state (id, data) VALUES (1, $json${
    "majors": [
        {"code": "TKJ",  "name": "Teknik Komputer dan Jaringan",   "workshopRoom": "Lab Jaringan 1",      "headOfMajor": "Ahmad Fauzi, S.Kom.",     "dudiPartnerCount": 6},
        {"code": "RPL",  "name": "Rekayasa Perangkat Lunak",       "workshopRoom": "Lab RPL 1",           "headOfMajor": "Dewi Lestari, S.T.",     "dudiPartnerCount": 8},
        {"code": "TBSM", "name": "Teknik dan Bisnis Sepeda Motor", "workshopRoom": "Bengkel Sepeda Motor", "headOfMajor": "Bambang Susilo, S.T.",   "dudiPartnerCount": 4},
        {"code": "AKL",  "name": "Akuntansi dan Keuangan Lembaga", "workshopRoom": "Lab Kantor 1",         "headOfMajor": "Rina Kusumawati, S.E.",  "dudiPartnerCount": 5}
    ],
    "pklPlacements": [
        {"companyName": "PT Cipta Digital Nusantara",   "industrySector": "Jasa IT"},
        {"companyName": "PT Nusantara Jaringan Sejahtera", "industrySector": "Telekomunikasi"},
        {"companyName": "PT Solusi Komputer Mandiri",  "industrySector": "Retail Komputer"},
        {"companyName": "PT Datakom Prima",            "industrySector": "Data Center"},
        {"companyName": "CV Bengkel Motor Jaya",       "industrySector": "Otomotif"},
        {"companyName": "PT Motorindo Group",          "industrySector": "Distribusi Kendaraan"},
        {"companyName": "PT Bank Artha Nusantara",     "industrySector": "Perbankan"},
        {"companyName": "PT Karya Akuntansi Mandiri",  "industrySector": "Jasa Keuangan"}
    ]
}$json$::jsonb)
ON CONFLICT (id) DO UPDATE SET data = app_state.data || EXCLUDED.data, updated_at = now();
