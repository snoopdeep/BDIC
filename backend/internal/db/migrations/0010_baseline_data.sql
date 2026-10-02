-- 0010_baseline_data.sql
-- The structure the school actually runs on, so the system is usable the first
-- time it starts rather than presenting empty dropdowns.
--
-- PLACEHOLDER VALUES. Everything marked TODO below is a guess that the school
-- must confirm. The affiliation number, UDISE code, phone numbers, office
-- hours, and Principal's name are not public information we could verify, so
-- they are written here as clearly-marked blanks rather than invented. They are
-- edited in the app under School settings, not by changing this file.

-- ---------------------------------------------------------------------------
-- The school
-- ---------------------------------------------------------------------------
INSERT INTO schools (
    name_en, name_hi, short_name,
    affiliation_no, udise_code, board,
    address_en, address_hi,
    village, district, state, pincode,
    phone_primary, phone_secondary, email,
    office_hours_en, office_hours_hi,
    principal_name,
    instagram_url, facebook_url, google_place_url
) VALUES (
    'Bhagwan Das Inter College',
    'भगवान दास इंटर कॉलेज',
    'BDIC',
    NULL,                                        -- TODO school: UP Board affiliation number
    NULL,                                        -- TODO school: UDISE code
    'UP Board (UPMSP)',
    'Ekauna, Chandauli, Uttar Pradesh',
    'एकौना, चंदौली, उत्तर प्रदेश',
    'Ekauna',
    'Chandauli',
    'Uttar Pradesh',
    NULL,                                        -- TODO school: PIN code
    NULL,                                        -- TODO school: office phone number
    NULL,                                        -- TODO school: alternate phone number
    NULL,                                        -- TODO school: official email address
    'Monday to Saturday, 8:00 AM to 2:00 PM',    -- TODO school: confirm office hours
    'सोमवार से शनिवार, सुबह 8:00 से दोपहर 2:00 बजे तक',
    NULL,                                        -- TODO school: Principal's name
    'https://www.instagram.com/bdic1146/',
    'https://www.facebook.com/groups/1206203054416295/',
    NULL                                         -- TODO school: Google Business listing link
);

-- ---------------------------------------------------------------------------
-- Academic session
-- ---------------------------------------------------------------------------
INSERT INTO academic_sessions (school_id, name, start_date, end_date, is_current)
SELECT id, '2026-27', DATE '2026-04-01', DATE '2027-03-31', true FROM schools;

-- ---------------------------------------------------------------------------
-- Intermediate streams
-- ---------------------------------------------------------------------------
INSERT INTO streams (code, name_en, name_hi, sort_order) VALUES
    ('SCI', 'Science',  'विज्ञान', 1),
    ('COM', 'Commerce', 'वाणिज्य', 2),
    ('ART', 'Arts',     'कला',     3);

-- ---------------------------------------------------------------------------
-- Classes. BDIC is an Inter College, so classes 9 to 12, with streams from 11.
-- ---------------------------------------------------------------------------
INSERT INTO classes (school_id, code, name_en, name_hi, level, has_stream, sort_order)
SELECT id, 'IX',  'Class 9',  'कक्षा 9',  9,  false, 1 FROM schools
UNION ALL
SELECT id, 'X',   'Class 10', 'कक्षा 10', 10, false, 2 FROM schools
UNION ALL
SELECT id, 'XI',  'Class 11', 'कक्षा 11', 11, true,  3 FROM schools
UNION ALL
SELECT id, 'XII', 'Class 12', 'कक्षा 12', 12, true,  4 FROM schools;

-- Sections. Two per class for High School; one per stream for Intermediate,
-- which is how a school this size is normally divided.
INSERT INTO sections (class_id, stream_id, name, capacity, sort_order)
SELECT c.id, NULL::uuid, 'A', 60, 1
  FROM classes c WHERE c.code IN ('IX', 'X')
UNION ALL
SELECT c.id, NULL::uuid, 'B', 60, 2
  FROM classes c WHERE c.code IN ('IX', 'X')
UNION ALL
SELECT c.id, s.id, s.name_en, 60, s.sort_order
  FROM classes c CROSS JOIN streams s
 WHERE c.code IN ('XI', 'XII');

-- ---------------------------------------------------------------------------
-- Subjects
-- ---------------------------------------------------------------------------
INSERT INTO subjects (school_id, code, name_en, name_hi, is_language, sort_order)
SELECT id, 'HIN', 'Hindi',             'हिंदी',            true,  1  FROM schools
UNION ALL SELECT id, 'ENG', 'English',           'अंग्रेज़ी',         true,  2  FROM schools
UNION ALL SELECT id, 'SAN', 'Sanskrit',          'संस्कृत',          true,  3  FROM schools
UNION ALL SELECT id, 'MAT', 'Mathematics',       'गणित',            false, 4  FROM schools
UNION ALL SELECT id, 'SCI', 'Science',           'विज्ञान',          false, 5  FROM schools
UNION ALL SELECT id, 'SST', 'Social Science',    'सामाजिक विज्ञान',  false, 6  FROM schools
UNION ALL SELECT id, 'PHY', 'Physics',           'भौतिक विज्ञान',    false, 7  FROM schools
UNION ALL SELECT id, 'CHE', 'Chemistry',         'रसायन विज्ञान',    false, 8  FROM schools
UNION ALL SELECT id, 'BIO', 'Biology',           'जीव विज्ञान',      false, 9  FROM schools
UNION ALL SELECT id, 'ACC', 'Accountancy',       'लेखाशास्त्र',      false, 10 FROM schools
UNION ALL SELECT id, 'BST', 'Business Studies',  'व्यवसाय अध्ययन',   false, 11 FROM schools
UNION ALL SELECT id, 'ECO', 'Economics',         'अर्थशास्त्र',      false, 12 FROM schools
UNION ALL SELECT id, 'HIS', 'History',           'इतिहास',          false, 13 FROM schools
UNION ALL SELECT id, 'GEO', 'Geography',         'भूगोल',           false, 14 FROM schools
UNION ALL SELECT id, 'POL', 'Political Science', 'राजनीति विज्ञान',  false, 15 FROM schools
UNION ALL SELECT id, 'SOC', 'Sociology',         'समाजशास्त्र',      false, 16 FROM schools
UNION ALL SELECT id, 'HSC', 'Home Science',      'गृह विज्ञान',      false, 17 FROM schools
UNION ALL SELECT id, 'PED', 'Physical Education','शारीरिक शिक्षा',   false, 18 FROM schools
UNION ALL SELECT id, 'CSC', 'Computer Science',  'कंप्यूटर विज्ञान', false, 19 FROM schools
UNION ALL SELECT id, 'ART', 'Drawing and Art',   'चित्रकला',         false, 20 FROM schools;

-- ---------------------------------------------------------------------------
-- Which subjects each class and stream runs.
-- TODO school: confirm the elective subjects actually offered, and the weekly
-- period count per subject. These are the common UP Board pattern, not BDIC's
-- confirmed timetable.
-- ---------------------------------------------------------------------------

-- High School: the five compulsory subjects, plus Sanskrit as an elective.
INSERT INTO class_subjects (session_id, class_id, stream_id, subject_id, is_compulsory, weekly_periods)
SELECT sess.id, c.id, NULL, sub.id, true, 6
  FROM academic_sessions sess
  CROSS JOIN classes c
  CROSS JOIN subjects sub
 WHERE sess.is_current
   AND c.code IN ('IX', 'X')
   AND sub.code IN ('HIN', 'ENG', 'MAT', 'SCI', 'SST');

INSERT INTO class_subjects (session_id, class_id, stream_id, subject_id, is_compulsory, weekly_periods)
SELECT sess.id, c.id, NULL, sub.id, false, 3
  FROM academic_sessions sess
  CROSS JOIN classes c
  CROSS JOIN subjects sub
 WHERE sess.is_current
   AND c.code IN ('IX', 'X')
   AND sub.code IN ('SAN', 'PED', 'ART');

-- Intermediate: Hindi and English are compulsory in every stream.
INSERT INTO class_subjects (session_id, class_id, stream_id, subject_id, is_compulsory, weekly_periods)
SELECT sess.id, c.id, str.id, sub.id, true, 5
  FROM academic_sessions sess
  CROSS JOIN classes c
  CROSS JOIN streams str
  CROSS JOIN subjects sub
 WHERE sess.is_current
   AND c.code IN ('XI', 'XII')
   AND sub.code IN ('HIN', 'ENG');

-- Science stream.
INSERT INTO class_subjects (session_id, class_id, stream_id, subject_id, is_compulsory, weekly_periods)
SELECT sess.id, c.id, str.id, sub.id, true, 6
  FROM academic_sessions sess
  CROSS JOIN classes c
  CROSS JOIN streams str
  CROSS JOIN subjects sub
 WHERE sess.is_current
   AND c.code IN ('XI', 'XII')
   AND str.code = 'SCI'
   AND sub.code IN ('PHY', 'CHE');

INSERT INTO class_subjects (session_id, class_id, stream_id, subject_id, is_compulsory, weekly_periods)
SELECT sess.id, c.id, str.id, sub.id, false, 6
  FROM academic_sessions sess
  CROSS JOIN classes c
  CROSS JOIN streams str
  CROSS JOIN subjects sub
 WHERE sess.is_current
   AND c.code IN ('XI', 'XII')
   AND str.code = 'SCI'
   AND sub.code IN ('BIO', 'MAT', 'CSC');

-- Commerce stream.
INSERT INTO class_subjects (session_id, class_id, stream_id, subject_id, is_compulsory, weekly_periods)
SELECT sess.id, c.id, str.id, sub.id, true, 6
  FROM academic_sessions sess
  CROSS JOIN classes c
  CROSS JOIN streams str
  CROSS JOIN subjects sub
 WHERE sess.is_current
   AND c.code IN ('XI', 'XII')
   AND str.code = 'COM'
   AND sub.code IN ('ACC', 'BST', 'ECO');

-- Arts stream.
INSERT INTO class_subjects (session_id, class_id, stream_id, subject_id, is_compulsory, weekly_periods)
SELECT sess.id, c.id, str.id, sub.id, true, 6
  FROM academic_sessions sess
  CROSS JOIN classes c
  CROSS JOIN streams str
  CROSS JOIN subjects sub
 WHERE sess.is_current
   AND c.code IN ('XI', 'XII')
   AND str.code = 'ART'
   AND sub.code IN ('HIS', 'GEO', 'POL');

INSERT INTO class_subjects (session_id, class_id, stream_id, subject_id, is_compulsory, weekly_periods)
SELECT sess.id, c.id, str.id, sub.id, false, 5
  FROM academic_sessions sess
  CROSS JOIN classes c
  CROSS JOIN streams str
  CROSS JOIN subjects sub
 WHERE sess.is_current
   AND c.code IN ('XI', 'XII')
   AND str.code = 'ART'
   AND sub.code IN ('SOC', 'HSC', 'ECO');

-- ---------------------------------------------------------------------------
-- The school day.
-- TODO school: confirm the period timings and where the break falls.
-- ---------------------------------------------------------------------------
INSERT INTO periods (school_id, position, name_en, name_hi, start_time, end_time, is_break)
SELECT id, 1, 'Period 1',  'कालांश 1', TIME '08:00', TIME '08:45', false FROM schools
UNION ALL SELECT id, 2, 'Period 2',  'कालांश 2', TIME '08:45', TIME '09:30', false FROM schools
UNION ALL SELECT id, 3, 'Period 3',  'कालांश 3', TIME '09:30', TIME '10:15', false FROM schools
UNION ALL SELECT id, 4, 'Break',     'अवकाश',    TIME '10:15', TIME '10:45', true  FROM schools
UNION ALL SELECT id, 5, 'Period 4',  'कालांश 4', TIME '10:45', TIME '11:30', false FROM schools
UNION ALL SELECT id, 6, 'Period 5',  'कालांश 5', TIME '11:30', TIME '12:15', false FROM schools
UNION ALL SELECT id, 7, 'Period 6',  'कालांश 6', TIME '12:15', TIME '13:00', false FROM schools
UNION ALL SELECT id, 8, 'Period 7',  'कालांश 7', TIME '13:00', TIME '13:45', false FROM schools;

-- ---------------------------------------------------------------------------
-- Grading scale.
-- TODO school: confirm these boundaries against the report card BDIC issues.
-- ---------------------------------------------------------------------------
INSERT INTO grading_scales (session_id, grade, min_percent, max_percent, remark_en, remark_hi, sort_order)
SELECT id, 'A+', 90.00, 100.00, 'Outstanding',    'उत्कृष्ट',      1 FROM academic_sessions WHERE is_current
UNION ALL SELECT id, 'A',  75.00, 89.99,  'Very good',      'बहुत अच्छा',    2 FROM academic_sessions WHERE is_current
UNION ALL SELECT id, 'B',  60.00, 74.99,  'Good',           'अच्छा',        3 FROM academic_sessions WHERE is_current
UNION ALL SELECT id, 'C',  45.00, 59.99,  'Satisfactory',   'संतोषजनक',     4 FROM academic_sessions WHERE is_current
UNION ALL SELECT id, 'D',  33.00, 44.99,  'Needs work',     'सुधार आवश्यक',  5 FROM academic_sessions WHERE is_current
UNION ALL SELECT id, 'E',  0.00,  32.99,  'Not passed',     'अनुत्तीर्ण',    6 FROM academic_sessions WHERE is_current;

-- ---------------------------------------------------------------------------
-- Fee heads.
-- TODO school: the amounts live in fee_plans, which the school fills in from
-- its own fee structure. Only the head names are set up here.
-- ---------------------------------------------------------------------------
INSERT INTO fee_heads (school_id, code, name_en, name_hi, is_recurring, sort_order)
SELECT id, 'ADMISSION',   'Admission Fee',   'प्रवेश शुल्क',      false, 1 FROM schools
UNION ALL SELECT id, 'TUITION',     'Tuition Fee',     'शिक्षण शुल्क',      true,  2 FROM schools
UNION ALL SELECT id, 'EXAM',        'Examination Fee', 'परीक्षा शुल्क',     false, 3 FROM schools
UNION ALL SELECT id, 'DEVELOPMENT', 'Development Fee', 'विकास शुल्क',       false, 4 FROM schools
UNION ALL SELECT id, 'COMPUTER',    'Computer Fee',    'कंप्यूटर शुल्क',    true,  5 FROM schools
UNION ALL SELECT id, 'LAB',         'Laboratory Fee',  'प्रयोगशाला शुल्क',  false, 6 FROM schools
UNION ALL SELECT id, 'LIBRARY',     'Library Fee',     'पुस्तकालय शुल्क',   false, 7 FROM schools
UNION ALL SELECT id, 'SPORTS',      'Sports Fee',      'खेल शुल्क',         false, 8 FROM schools;

-- Concession types the school is likely to need. Each one records who approved
-- it and why, so the fee reports can show what was given away and by whom.
INSERT INTO concession_types (school_id, code, name_en, name_hi, kind, default_value, requires_approval)
SELECT id, 'SCHOLARSHIP', 'Scholarship',           'छात्रवृत्ति',            'PERCENT', 100.00, true FROM schools
UNION ALL SELECT id, 'SIBLING',     'Sibling discount',      'भाई-बहन छूट',            'PERCENT', 10.00,  true FROM schools
UNION ALL SELECT id, 'STAFF_WARD',  'Staff ward concession', 'कर्मचारी संतान छूट',     'PERCENT', 50.00,  true FROM schools
UNION ALL SELECT id, 'MERIT',       'Merit concession',      'मेधा छूट',               'PERCENT', 25.00,  true FROM schools
UNION ALL SELECT id, 'HARDSHIP',    'Hardship waiver',       'आर्थिक कठिनाई छूट',      'PERCENT', 50.00,  true FROM schools;

-- ---------------------------------------------------------------------------
-- Number series. Admission numbers, receipts and certificates are issued from
-- these counters so the formats stay gapless and predictable.
-- TODO school: confirm the prefixes and the number the receipt series should
-- continue from, so the new system does not restart at 1 alongside the old
-- receipt book.
-- ---------------------------------------------------------------------------
INSERT INTO number_series (scope, period, prefix, next_value, pad_width) VALUES
    ('ADMISSION_NO',     '2026-27', 'BDIC/2026/', 1, 4),
    ('APPLICATION_NO',   '2026-27', 'APP/2026/',  1, 4),
    ('RECEIPT_NO',       '2026-27', 'RCP/2026/',  1, 5),
    ('INVOICE_NO',       '2026-27', 'INV/2026/',  1, 5),
    ('CERT_BONAFIDE',    '2026-27', 'BON/2026/',  1, 4),
    ('CERT_CHARACTER',   '2026-27', 'CHR/2026/',  1, 4),
    ('CERT_TRANSFER',    '2026-27', 'TC/2026/',   1, 4),
    ('CERT_ATTENDANCE',  '2026-27', 'ATT/2026/',  1, 4);

-- ---------------------------------------------------------------------------
-- Documents an applicant must upload.
-- A NULL class_id means the requirement applies to every class.
-- TODO school: confirm this list, and whether any differ by class.
-- ---------------------------------------------------------------------------
INSERT INTO admission_required_documents (class_id, doc_type, label_en, label_hi, mandatory, sort_order) VALUES
    (NULL, 'BIRTH_CERTIFICATE', 'Birth certificate',              'जन्म प्रमाण पत्र',           true,  1),
    (NULL, 'PREVIOUS_MARKSHEET','Last class mark sheet',          'पिछली कक्षा की अंकतालिका',   true,  2),
    (NULL, 'TRANSFER_CERT',     'Transfer certificate',           'स्थानांतरण प्रमाण पत्र',     true,  3),
    (NULL, 'AADHAAR',           'Aadhaar card',                   'आधार कार्ड',                 true,  4),
    (NULL, 'PHOTO',             'Passport size photograph',       'पासपोर्ट आकार फोटो',         true,  5),
    (NULL, 'CASTE_CERT',        'Caste certificate, if claiming', 'जाति प्रमाण पत्र, यदि लागू', false, 6),
    (NULL, 'INCOME_CERT',       'Income certificate, if claiming','आय प्रमाण पत्र, यदि लागू',   false, 7);

-- ---------------------------------------------------------------------------
-- School settings the office can change without a deploy.
-- ---------------------------------------------------------------------------
INSERT INTO settings (key, value, description) VALUES
    ('site.defaultLocale', '"hi"'::jsonb,
     'Language a first-time visitor sees. TODO school: confirm Hindi or English.'),
    ('attendance.statuses', '["PRESENT","ABSENT","LATE","LEAVE","MEDICAL"]'::jsonb,
     'Attendance statuses in use. TODO school: confirm which are wanted.'),
    ('attendance.lowThresholdPercent', '75'::jsonb,
     'Below this, a student is flagged to the class teacher. TODO school: confirm.'),
    ('attendance.absenceAlertsEnabled', 'false'::jsonb,
     'Off until the school opens an SMS account. Drafts are written either way.'),
    ('attendance.correctionWindowDays', '1'::jsonb,
     'A teacher may edit freely for this many days; after that it needs approval.'),
    ('homework.allowOnlineSubmission', 'false'::jsonb,
     'TODO school: decide whether students upload work in the first release.'),
    ('messaging.parentTeacherEnabled', 'false'::jsonb,
     'TODO school: decide whether parent-teacher messaging is on at launch.'),
    ('messaging.allowedHours', '{"from":"08:00","to":"18:00"}'::jsonb,
     'Hours during which a parent may message a teacher.'),
    ('notices.approvalRequired', 'true'::jsonb,
     'Whether a notice needs the Principal before it publishes.'),
    ('gallery.requirePhotoApproval', 'true'::jsonb,
     'Photographs containing students are held until approved. Leave this on.'),
    ('fees.onlinePaymentEnabled', 'false'::jsonb,
     'Off until the school signs with a payment gateway. Office collection works.'),
    ('exams.showRank', 'false'::jsonb,
     'TODO school: decide whether the report card shows a class rank.');

-- ---------------------------------------------------------------------------
-- Public website pages. The body text is a placeholder in both languages, so
-- every page renders correctly and the school can see what it needs to write.
-- ---------------------------------------------------------------------------
INSERT INTO site_pages (slug, title_en, title_hi, body_en, body_hi, sort_order) VALUES
    ('about', 'About the School', 'विद्यालय के बारे में',
     'TODO school: a few paragraphs on the school''s history and character.',
     'TODO विद्यालय: विद्यालय के इतिहास और स्वरूप पर कुछ अनुच्छेद।', 1),
    ('principal-message', 'Principal''s Message', 'प्रधानाचार्य का संदेश',
     'TODO school: the Principal''s message to students and parents.',
     'TODO विद्यालय: विद्यार्थियों और अभिभावकों के लिए प्रधानाचार्य का संदेश।', 2),
    ('vision-mission', 'Vision and Mission', 'दृष्टि और लक्ष्य',
     'TODO school: the school''s vision and mission.',
     'TODO विद्यालय: विद्यालय की दृष्टि और लक्ष्य।', 3),
    ('admission-info', 'Admission Information', 'प्रवेश जानकारी',
     'TODO school: who may apply, eligibility, important dates, and the fee summary.',
     'TODO विद्यालय: कौन आवेदन कर सकता है, पात्रता, महत्वपूर्ण तिथियाँ और शुल्क विवरण।', 4),
    ('privacy', 'Privacy Policy', 'गोपनीयता नीति',
     'TODO school: how student information and photographs are handled.',
     'TODO विद्यालय: विद्यार्थी की जानकारी और फोटो का उपयोग कैसे किया जाता है।', 5),
    ('terms', 'Terms of Use', 'उपयोग की शर्तें',
     'TODO school: terms for using the school portal.',
     'TODO विद्यालय: विद्यालय पोर्टल के उपयोग की शर्तें।', 6);

-- Facilities shown on the public site. Photographs are attached later, once
-- the school provides approved images.
INSERT INTO facilities (title_en, title_hi, description_en, description_hi, icon, sort_order) VALUES
    ('Science Laboratory', 'विज्ञान प्रयोगशाला',
     'Physics, Chemistry and Biology practical work for Intermediate students.',
     'इंटरमीडिएट विद्यार्थियों के लिए भौतिकी, रसायन और जीव विज्ञान का प्रायोगिक कार्य।', 'flask', 1),
    ('Library', 'पुस्तकालय',
     'Reference books, textbooks and previous-year question papers.',
     'संदर्भ पुस्तकें, पाठ्यपुस्तकें और पिछले वर्षों के प्रश्नपत्र।', 'book', 2),
    ('Computer Room', 'कंप्यूटर कक्ष',
     'Computer practice for Computer Science and Informatics students.',
     'कंप्यूटर विज्ञान के विद्यार्थियों के लिए अभ्यास कक्ष।', 'monitor', 3),
    ('Playground', 'खेल का मैदान',
     'Cricket, kabaddi, volleyball and athletics.',
     'क्रिकेट, कबड्डी, वॉलीबॉल और एथलेटिक्स।', 'trophy', 4),
    ('Classrooms', 'कक्षा-कक्ष',
     'Ventilated classrooms with seating for the full class strength.',
     'पूरी कक्षा के लिए बैठने की व्यवस्था वाले हवादार कक्षा-कक्ष।', 'chalkboard', 5),
    ('Drinking Water', 'पेयजल',
     'Clean drinking water available throughout the school day.',
     'विद्यालय समय में स्वच्छ पेयजल की उपलब्धता।', 'droplet', 6);
