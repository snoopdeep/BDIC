-- BDIC local demonstration data
--
-- This file contains only fictional records. It is safe to run repeatedly:
-- each row has a DEMO-* identifier or is protected by a NOT EXISTS check.
-- Do not run it against a production school database.

BEGIN;

UPDATE schools
   SET phone_primary = '9000000000',
       email = 'demo.office@bdic.local',
       principal_name = 'Demo Principal',
       pincode = '232101',
       updated_at = now()
 WHERE id = (SELECT id FROM schools LIMIT 1);

INSERT INTO site_pages (slug, title_en, title_hi, body_en, body_hi, sort_order)
VALUES
  ('about', 'About BDIC', 'बीडीआईसी के बारे में', 'Demonstration content for the public school profile.', 'विद्यालय प्रोफ़ाइल के लिए प्रदर्शन सामग्री।', 1),
  ('admissions', 'Admissions', 'प्रवेश', 'Demonstration information about admissions and documents.', 'प्रवेश और दस्तावेज़ों की प्रदर्शन जानकारी।', 2),
  ('privacy', 'Privacy', 'गोपनीयता', 'Demonstration privacy information.', 'प्रदर्शन गोपनीयता जानकारी।', 3),
  ('terms', 'Terms of use', 'उपयोग की शर्तें', 'Demonstration portal terms.', 'प्रदर्शन पोर्टल नियम।', 4)
ON CONFLICT (slug) DO UPDATE
  SET title_en = EXCLUDED.title_en, title_hi = EXCLUDED.title_hi,
      body_en = EXCLUDED.body_en, body_hi = EXCLUDED.body_hi, updated_at = now();

INSERT INTO gallery_albums (slug, title_en, title_hi, category, description_en, description_hi, event_date, is_public, sort_order)
VALUES
  ('demo-campus-learning', 'Campus learning spaces', 'परिसर के अध्ययन क्षेत्र', 'CAMPUS', 'Fictional demo album ready for consent-checked campus photographs.', 'सहमति-युक्त परिसर चित्रों के लिए काल्पनिक डेमो एल्बम।', CURRENT_DATE - interval '20 days', true, 1),
  ('demo-science-day', 'Science Learning Day', 'विज्ञान अधिगम दिवस', 'EVENT', 'Fictional demo album for a Class 9 learning activity.', 'कक्षा 9 की काल्पनिक डेमो गतिविधि का एल्बम।', CURRENT_DATE - interval '7 days', true, 2)
ON CONFLICT (slug) DO UPDATE
  SET title_en = EXCLUDED.title_en, title_hi = EXCLUDED.title_hi, category = EXCLUDED.category,
      description_en = EXCLUDED.description_en, description_hi = EXCLUDED.description_hi,
      event_date = EXCLUDED.event_date, is_public = EXCLUDED.is_public, sort_order = EXCLUDED.sort_order;

INSERT INTO staff (employee_code, full_name_en, full_name_hi, designation_en, designation_hi, department, qualification, phone, email, show_on_website)
SELECT demo.employee_code, demo.full_name_en, demo.full_name_hi, demo.designation_en, demo.designation_hi, demo.department, demo.qualification, demo.phone, demo.email, true
  FROM (VALUES
    ('DEMO-T002', 'Anita Verma', 'अनीता वर्मा', 'Hindi Teacher', 'हिंदी शिक्षिका', 'Languages', 'M.A., B.Ed.', '9000000012', 'anita.demo@bdic.local'),
    ('DEMO-T003', 'Rakesh Singh', 'राकेश सिंह', 'Science Teacher', 'विज्ञान शिक्षक', 'Science', 'M.Sc., B.Ed.', '9000000013', 'rakesh.demo@bdic.local'),
    ('DEMO-T004', 'Kavita Yadav', 'कविता यादव', 'Mathematics Teacher', 'गणित शिक्षिका', 'Mathematics', 'B.Sc., B.Ed.', '9000000014', 'kavita.demo@bdic.local')
  ) AS demo(employee_code, full_name_en, full_name_hi, designation_en, designation_hi, department, qualification, phone, email)
 WHERE NOT EXISTS (SELECT 1 FROM staff s WHERE s.employee_code = demo.employee_code);

INSERT INTO students (admission_no, full_name_en, full_name_hi, gender, category, father_name, mother_name, village, district, state, pincode, status)
SELECT demo.admission_no, demo.full_name_en, demo.full_name_hi, demo.gender, 'GEN', demo.father_name, demo.mother_name, 'Ekauna', 'Chandauli', 'Uttar Pradesh', '232101', 'ACTIVE'
  FROM (VALUES
    ('DEMO-IX-002', 'Aarav Mishra', 'आरव मिश्रा', 'MALE', 'Sanjay Mishra', 'Poonam Mishra'),
    ('DEMO-IX-003', 'Diya Patel', 'दिया पटेल', 'FEMALE', 'Manoj Patel', 'Neha Patel'),
    ('DEMO-IX-004', 'Krishna Gupta', 'कृष्ण गुप्ता', 'MALE', 'Vijay Gupta', 'Seema Gupta'),
    ('DEMO-IX-005', 'Sana Khan', 'सना खान', 'FEMALE', 'Imran Khan', 'Farah Khan'),
    ('DEMO-IX-006', 'Vivek Rai', 'विवेक राय', 'MALE', 'Dinesh Rai', 'Rekha Rai')
  ) AS demo(admission_no, full_name_en, full_name_hi, gender, father_name, mother_name)
 WHERE NOT EXISTS (SELECT 1 FROM students s WHERE s.admission_no = demo.admission_no);

INSERT INTO guardians (full_name_en, full_name_hi, relation, phone, email)
SELECT demo.full_name_en, demo.full_name_hi, 'FATHER', demo.phone, demo.email
  FROM (VALUES
    ('Sanjay Mishra', 'संजय मिश्रा', '9000000102', 'sanjay.demo@bdic.local'),
    ('Manoj Patel', 'मनोज पटेल', '9000000103', 'manoj.demo@bdic.local'),
    ('Vijay Gupta', 'विजय गुप्ता', '9000000104', 'vijay.demo@bdic.local'),
    ('Imran Khan', 'इमरान खान', '9000000105', 'imran.demo@bdic.local'),
    ('Dinesh Rai', 'दिनेश राय', '9000000106', 'dinesh.demo@bdic.local')
  ) AS demo(full_name_en, full_name_hi, phone, email)
 WHERE NOT EXISTS (SELECT 1 FROM guardians g WHERE g.phone = demo.phone);

INSERT INTO student_guardians (student_id, guardian_id, is_primary)
SELECT s.id, g.id, true
  FROM students s
  JOIN guardians g ON g.full_name_en = s.father_name
 WHERE s.admission_no LIKE 'DEMO-IX-%'
ON CONFLICT (student_id, guardian_id) DO NOTHING;

INSERT INTO enrollments (student_id, session_id, class_id, section_id, roll_no)
SELECT st.id, sess.id, cls.id, sec.id, row_number() OVER (ORDER BY st.admission_no) + 1
  FROM students st
 CROSS JOIN (SELECT id FROM academic_sessions WHERE is_current LIMIT 1) sess
 CROSS JOIN (SELECT id FROM classes WHERE code = 'IX' LIMIT 1) cls
 CROSS JOIN (SELECT id FROM sections WHERE class_id = (SELECT id FROM classes WHERE code = 'IX' LIMIT 1) AND name = 'A' LIMIT 1) sec
 WHERE st.admission_no LIKE 'DEMO-IX-%'
ON CONFLICT (student_id, session_id) DO NOTHING;

INSERT INTO attendance (session_id, student_id, section_id, date, status, note)
SELECT sess.id, st.id, sec.id, days.day::date,
       CASE WHEN st.admission_no = 'DEMO-IX-005' AND extract(dow FROM days.day) IN (2, 5) THEN 'ABSENT'
            WHEN st.admission_no = 'DEMO-IX-003' AND extract(dow FROM days.day) = 4 THEN 'LATE'
            ELSE 'PRESENT' END,
       CASE WHEN st.admission_no = 'DEMO-IX-005' AND extract(dow FROM days.day) IN (2, 5) THEN 'Sample absence' ELSE NULL END
  FROM students st
 CROSS JOIN (SELECT id FROM academic_sessions WHERE is_current LIMIT 1) sess
 CROSS JOIN (SELECT id FROM sections WHERE class_id = (SELECT id FROM classes WHERE code = 'IX' LIMIT 1) AND name = 'A' LIMIT 1) sec
 CROSS JOIN generate_series(CURRENT_DATE - interval '8 days', CURRENT_DATE - interval '1 day', interval '1 day') days(day)
 WHERE st.admission_no LIKE 'DEMO-IX-%'
ON CONFLICT (student_id, date) DO NOTHING;

INSERT INTO attendance_submissions (session_id, section_id, date, present, absent, other)
SELECT sess.id, sec.id, a.date,
       count(*) FILTER (WHERE a.status = 'PRESENT'),
       count(*) FILTER (WHERE a.status = 'ABSENT'),
       count(*) FILTER (WHERE a.status NOT IN ('PRESENT', 'ABSENT'))
  FROM attendance a
 CROSS JOIN (SELECT id FROM academic_sessions WHERE is_current LIMIT 1) sess
 CROSS JOIN (SELECT id FROM sections WHERE class_id = (SELECT id FROM classes WHERE code = 'IX' LIMIT 1) AND name = 'A' LIMIT 1) sec
 WHERE a.section_id = sec.id AND a.date >= CURRENT_DATE - interval '8 days'
 GROUP BY sess.id, sec.id, a.date
ON CONFLICT (section_id, date) DO UPDATE
  SET present = EXCLUDED.present, absent = EXCLUDED.absent, other = EXCLUDED.other, submitted_at = now();

INSERT INTO timetable_entries (session_id, section_id, day_of_week, period_id, subject_id, staff_id, room)
SELECT sess.id, sec.id, demo.day_of_week, period.id, subject.id, teacher.id, demo.room
  FROM (VALUES
    (1, 1, 'HIN', 'DEMO-T002', 'Room 101'), (1, 2, 'ENG', 'TEST-T001', 'Room 101'),
    (1, 3, 'MAT', 'DEMO-T004', 'Room 101'), (2, 1, 'SCI', 'DEMO-T003', 'Science Lab'),
    (2, 2, 'HIN', 'DEMO-T002', 'Room 101'), (3, 1, 'MAT', 'DEMO-T004', 'Room 101'),
    (3, 2, 'ENG', 'TEST-T001', 'Room 101'), (4, 1, 'SCI', 'DEMO-T003', 'Science Lab'),
    (5, 1, 'HIN', 'DEMO-T002', 'Room 101'), (6, 1, 'MAT', 'DEMO-T004', 'Room 101')
  ) AS demo(day_of_week, period_position, subject_code, employee_code, room)
 CROSS JOIN (SELECT id FROM academic_sessions WHERE is_current LIMIT 1) sess
 CROSS JOIN (SELECT id FROM sections WHERE class_id = (SELECT id FROM classes WHERE code = 'IX' LIMIT 1) AND name = 'A' LIMIT 1) sec
 JOIN periods period ON period.position = demo.period_position
 JOIN subjects subject ON subject.code = demo.subject_code
 JOIN staff teacher ON teacher.employee_code = demo.employee_code
ON CONFLICT (session_id, section_id, day_of_week, period_id) DO NOTHING;

INSERT INTO homework (session_id, section_id, subject_id, staff_id, title, instructions, due_date, allow_upload, status)
SELECT sess.id, sec.id, sub.id, teacher.id, demo.title, demo.instructions, CURRENT_DATE + demo.due_offset, true, 'PUBLISHED'
  FROM (VALUES
    ('Read chapter 4 and answer questions 1–5', 'Read the Hindi lesson and write concise answers in your exercise book.', 3),
    ('Algebra practice worksheet', 'Complete the five linear-equation problems before the next mathematics class.', 5),
    ('Observe a simple science experiment', 'Record observations from the classroom demonstration in your notebook.', 7)
  ) AS demo(title, instructions, due_offset)
 CROSS JOIN (SELECT id FROM academic_sessions WHERE is_current LIMIT 1) sess
 CROSS JOIN (SELECT id FROM sections WHERE class_id = (SELECT id FROM classes WHERE code = 'IX' LIMIT 1) AND name = 'A' LIMIT 1) sec
 JOIN subjects sub ON sub.code = CASE WHEN demo.title LIKE 'Algebra%' THEN 'MAT' WHEN demo.title LIKE 'Observe%' THEN 'SCI' ELSE 'HIN' END
 JOIN staff teacher ON teacher.employee_code = CASE WHEN demo.title LIKE 'Algebra%' THEN 'DEMO-T004' WHEN demo.title LIKE 'Observe%' THEN 'DEMO-T003' ELSE 'DEMO-T002' END
 WHERE NOT EXISTS (SELECT 1 FROM homework h WHERE h.title = demo.title);

INSERT INTO study_materials (session_id, class_id, subject_id, title_en, title_hi, description, category, link_url)
SELECT sess.id, cls.id, sub.id, 'Demo Hindi revision notes', 'डेमो हिंदी पुनरावृत्ति नोट्स',
       'A fictional study-material record for reviewing the class library flow.',
       'NOTES', 'https://example.invalid/bdic-demo-hindi-notes'
  FROM (SELECT id FROM academic_sessions WHERE is_current LIMIT 1) sess
 CROSS JOIN (SELECT id FROM classes WHERE code = 'IX' LIMIT 1) cls
 CROSS JOIN (SELECT id FROM subjects WHERE code = 'HIN' LIMIT 1) sub
 WHERE NOT EXISTS (SELECT 1 FROM study_materials WHERE title_en = 'Demo Hindi revision notes');

INSERT INTO notices (title_en, title_hi, body_en, body_hi, audience_kind, audience_roles, requires_ack, is_public, status, publish_at)
SELECT demo.title_en, demo.title_hi, demo.body_en, demo.body_hi, 'WHOLE_SCHOOL', '{}', demo.requires_ack, true, 'PUBLISHED', now() - demo.age
  FROM (VALUES
    ('Welcome to the demonstration portal', 'डेमो पोर्टल में आपका स्वागत है', 'These fictional records are for reviewing the school-management workflow.', 'ये काल्पनिक रिकॉर्ड विद्यालय प्रबंधन प्रवाह की समीक्षा के लिए हैं।', false, interval '2 days'),
    ('Parent meeting: demonstration schedule', 'अभिभावक बैठक: डेमो समय-सारिणी', 'A sample parent-teacher meeting is listed in the Events page.', 'नमूना अभिभावक-शिक्षक बैठक इवेंट पृष्ठ पर सूचीबद्ध है।', true, interval '1 day')
  ) AS demo(title_en, title_hi, body_en, body_hi, requires_ack, age)
 WHERE NOT EXISTS (SELECT 1 FROM notices n WHERE n.title_en = demo.title_en);

INSERT INTO events (session_id, title_en, title_hi, description_en, description_hi, start_date, start_time, venue_en, venue_hi, category, is_public)
SELECT sess.id, demo.title_en, demo.title_hi, demo.description_en, demo.description_hi, CURRENT_DATE + demo.days_ahead, demo.start_time::time, demo.venue_en, demo.venue_hi, demo.category, true
  FROM (VALUES
    ('Parent–Teacher Meeting', 'अभिभावक-शिक्षक बैठक', 'Sample discussion of attendance and learning progress.', 'उपस्थिति और पढ़ाई की प्रगति पर नमूना चर्चा।', 5, '10:00', 'School Hall', 'विद्यालय सभागार', 'PTM'),
    ('Science Learning Day', 'विज्ञान अधिगम दिवस', 'A sample hands-on science activity for Class 9.', 'कक्षा 9 के लिए नमूना विज्ञान गतिविधि।', 12, '09:30', 'Science Laboratory', 'विज्ञान प्रयोगशाला', 'GENERAL')
  ) AS demo(title_en, title_hi, description_en, description_hi, days_ahead, start_time, venue_en, venue_hi, category)
 CROSS JOIN (SELECT id FROM academic_sessions WHERE is_current LIMIT 1) sess
 WHERE NOT EXISTS (SELECT 1 FROM events e WHERE e.title_en = demo.title_en);

INSERT INTO achievements (title_en, title_hi, student_name, year, category, detail_en, detail_hi, marks_or_rank, is_featured)
SELECT demo.title_en, demo.title_hi, demo.student_name, extract(year FROM CURRENT_DATE)::int, demo.category, demo.detail_en, demo.detail_hi, demo.marks_or_rank, true
  FROM (VALUES
    ('Sample mathematics achievement', 'नमूना गणित उपलब्धि', 'Diya Patel', 'ACADEMIC', 'Demonstration achievement for the public-site review.', 'सार्वजनिक साइट समीक्षा के लिए प्रदर्शन उपलब्धि।', 'Rank 1'),
    ('Sample science presentation', 'नमूना विज्ञान प्रस्तुति', 'Aarav Mishra', 'CULTURAL', 'Demonstration activity achievement.', 'प्रदर्शन गतिविधि उपलब्धि।', 'Certificate')
  ) AS demo(title_en, title_hi, student_name, category, detail_en, detail_hi, marks_or_rank)
 WHERE NOT EXISTS (SELECT 1 FROM achievements a WHERE a.title_en = demo.title_en);

INSERT INTO facilities (title_en, title_hi, description_en, description_hi, icon, sort_order, is_published)
SELECT demo.title_en, demo.title_hi, demo.description_en, demo.description_hi, demo.icon, demo.sort_order, true
  FROM (VALUES
    ('Science Laboratory', 'विज्ञान प्रयोगशाला', 'A demonstration description for practical science learning.', 'प्रायोगिक विज्ञान अधिगम के लिए प्रदर्शन विवरण।', 'flask', 1),
    ('Library', 'पुस्तकालय', 'A demonstration description for reading and reference resources.', 'पठन और संदर्भ संसाधनों के लिए प्रदर्शन विवरण।', 'book', 2),
    ('Sports Ground', 'खेल मैदान', 'A demonstration description for sports and assembly activities.', 'खेल और प्रार्थना सभा गतिविधियों के लिए प्रदर्शन विवरण।', 'trophy', 3)
  ) AS demo(title_en, title_hi, description_en, description_hi, icon, sort_order)
 WHERE NOT EXISTS (SELECT 1 FROM facilities f WHERE f.title_en = demo.title_en);

INSERT INTO admission_applications (application_no, session_id, applicant_name, gender, category, class_applied_id, father_name, guardian_name, guardian_relation, guardian_phone, state, lookup_token, status, submitted_at)
SELECT demo.application_no, sess.id, demo.applicant_name, demo.gender, 'GEN', cls.id, demo.father_name, demo.father_name, 'FATHER', demo.phone, 'Uttar Pradesh', 'demo-lookup-token-not-for-production', demo.status, now() - demo.age
  FROM (VALUES
    ('DEMO-APP-001', 'Nisha Singh', 'FEMALE', 'Rajesh Singh', '9000000201', 'UNDER_REVIEW', interval '3 days'),
    ('DEMO-APP-002', 'Aditya Kumar', 'MALE', 'Pradeep Kumar', '9000000202', 'TEST_SCHEDULED', interval '2 days'),
    ('DEMO-APP-003', 'Pihu Yadav', 'FEMALE', 'Ramesh Yadav', '9000000203', 'DOCUMENTS_NEEDED', interval '1 day')
  ) AS demo(application_no, applicant_name, gender, father_name, phone, status, age)
 CROSS JOIN (SELECT id FROM academic_sessions WHERE is_current LIMIT 1) sess
 CROSS JOIN (SELECT id FROM classes WHERE code = 'IX' LIMIT 1) cls
ON CONFLICT (application_no) DO NOTHING;

INSERT INTO exams (session_id, name_en, name_hi, term, weight_percent, status, published_at)
SELECT id, 'Demo Mid-Term Assessment', 'डेमो मध्यावधि मूल्यांकन', 'Mid-Term', 30, 'PUBLISHED', now()
  FROM academic_sessions WHERE is_current
ON CONFLICT (session_id, name_en) DO UPDATE SET status = 'PUBLISHED', published_at = now();

INSERT INTO exam_subjects (exam_id, class_id, subject_id, exam_date, start_time, max_marks, pass_marks)
SELECT exam.id, cls.id, sub.id, CURRENT_DATE + interval '14 days', time '10:00', 100, 33
  FROM exams exam
 CROSS JOIN (SELECT id FROM classes WHERE code = 'IX' LIMIT 1) cls
 CROSS JOIN (SELECT id FROM subjects WHERE code = 'HIN' LIMIT 1) sub
 WHERE exam.name_en = 'Demo Mid-Term Assessment'
   AND NOT EXISTS (SELECT 1 FROM exam_subjects es WHERE es.exam_id = exam.id AND es.class_id = cls.id AND es.subject_id = sub.id);

INSERT INTO marks (exam_subject_id, student_id, marks_obtained, special_status, remark)
SELECT es.id, st.id, 82, 'NONE', 'Demonstration mark'
  FROM exam_subjects es
  JOIN exams exam ON exam.id = es.exam_id AND exam.name_en = 'Demo Mid-Term Assessment'
  JOIN students st ON st.admission_no = 'TEST-S001'
ON CONFLICT (exam_subject_id, student_id) DO UPDATE SET marks_obtained = EXCLUDED.marks_obtained, remark = EXCLUDED.remark;

INSERT INTO academic_records (student_id, subject, academic_year, session, marks_obtained, max_marks, grade, record_type)
SELECT st.id, demo.subject, '2026-27', 'Mid-Term', demo.marks, 100, demo.grade, 'EXAM'
  FROM (VALUES ('Hindi', 82::numeric, 'A'), ('Mathematics', 88::numeric, 'A'), ('Science', 76::numeric, 'A')) AS demo(subject, marks, grade)
  JOIN students st ON st.admission_no = 'TEST-S001'
 WHERE NOT EXISTS (SELECT 1 FROM academic_records ar WHERE ar.student_id = st.id AND ar.subject = demo.subject AND ar.academic_year = '2026-27' AND ar.session = 'Mid-Term');

INSERT INTO invoices (invoice_no, student_id, session_id, installment_no, due_date, gross_paise, paid_paise, status)
SELECT 'DEMO-INV-002', st.id, sess.id, 2, CURRENT_DATE + interval '10 days', 180000, 0, 'DUE'
  FROM students st CROSS JOIN (SELECT id FROM academic_sessions WHERE is_current LIMIT 1) sess
 WHERE st.admission_no = 'DEMO-IX-002'
ON CONFLICT (invoice_no) DO NOTHING;

INSERT INTO invoice_items (invoice_id, fee_head_id, amount_paise)
SELECT inv.id, head.id, 180000 FROM invoices inv CROSS JOIN (SELECT id FROM fee_heads WHERE code = 'TUITION' LIMIT 1) head
 WHERE inv.invoice_no = 'DEMO-INV-002'
ON CONFLICT (invoice_id, fee_head_id) DO NOTHING;

INSERT INTO outbound_messages (channel, recipient, template_key, subject, body, status, related_type, related_id)
SELECT 'SMS', '9000000102', 'DEMO_ATTENDANCE_ALERT', 'Demo message', 'This is a local demonstration SMS draft. No message has been sent.', 'DRAFT', 'demo', 'DEMO-OUTBOUND-1'
 WHERE NOT EXISTS (SELECT 1 FROM outbound_messages WHERE related_id = 'DEMO-OUTBOUND-1');

COMMIT;
