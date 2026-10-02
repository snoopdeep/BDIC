-- 0012_lkg_to_12th_classes.sql
-- Allow pre-primary levels (-5 to 0) in classes_level_range constraint, then add LKG, UKG, Nursery, and Classes 1 to 8.

ALTER TABLE classes DROP CONSTRAINT IF EXISTS classes_level_range;
ALTER TABLE classes ADD CONSTRAINT classes_level_range CHECK (level BETWEEN -5 AND 12);

INSERT INTO classes (school_id, code, name_en, name_hi, level, has_stream, sort_order)
SELECT s.id, c.code, c.name_en, c.name_hi, c.level, c.has_stream, c.sort_order
FROM schools s
CROSS JOIN (
    VALUES
        ('LKG',  'LKG',       'एल.के.जी.', -2, false, 1),
        ('UKG',  'UKG',       'यू.के.जी.', -1, false, 2),
        ('NUR',  'Nursery',   'नर्सरी',    0,  false, 3),
        ('I',    'Class 1',   'कक्षा 1',   1,  false, 4),
        ('II',   'Class 2',   'कक्षा 2',   2,  false, 5),
        ('III',  'Class 3',   'कक्षा 3',   3,  false, 6),
        ('IV',   'Class 4',   'कक्षा 4',   4,  false, 7),
        ('V',    'Class 5',   'कक्षा 5',   5,  false, 8),
        ('VI',   'Class 6',   'कक्षा 6',   6,  false, 9),
        ('VII',  'Class 7',   'कक्षा 7',   7,  false, 10),
        ('VIII', 'Class 8',   'कक्षा 8',   8,  false, 11)
) AS c(code, name_en, name_hi, level, has_stream, sort_order)
ON CONFLICT (school_id, code) DO NOTHING;

-- Sections A & B for the newly added classes
INSERT INTO sections (class_id, stream_id, name, capacity, sort_order)
SELECT c.id, NULL::uuid, sec.name, 60, sec.sort_order
FROM classes c
CROSS JOIN (
    VALUES ('A', 1), ('B', 2)
) AS sec(name, sort_order)
WHERE c.code IN ('LKG', 'UKG', 'NUR', 'I', 'II', 'III', 'IV', 'V', 'VI', 'VII', 'VIII')
ON CONFLICT (class_id, name) DO NOTHING;
