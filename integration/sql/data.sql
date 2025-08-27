-- Sample data for simple-query-server example database
-- This file populates the tables with test data

-- Insert sample users
INSERT INTO users (name, email, status, active) VALUES
    ('Alice Smith', 'alice.smith@example.com', 'active', true),
    ('Bob Johnson', 'bob.johnson@example.com', 'active', true),
    ('Carol Williams', 'carol.williams@example.com', 'active', true),
    ('David Brown', 'david.brown@example.com', 'inactive', false),
    ('Eva Davis', 'eva.davis@example.com', 'active', true),
    ('Frank Miller', 'frank.miller@example.com', 'active', true),
    ('Grace Wilson', 'grace.wilson@example.com', 'suspended', false),
    ('Henry Moore', 'henry.moore@example.com', 'active', true),
    ('Isabella Taylor', 'isabella.taylor@example.com', 'active', true),
    ('Jack Anderson', 'jack.anderson@example.com', 'active', true),
    ('Kate Thomas', 'kate.thomas@example.com', 'active', true),
    ('Liam Jackson', 'liam.jackson@example.com', 'inactive', false),
    ('Mia White', 'mia.white@example.com', 'active', true),
    ('Noah Harris', 'noah.harris@example.com', 'active', true),
    ('Olivia Martin', 'olivia.martin@example.com', 'active', true),
    ('Paul Thompson', 'paul.thompson@example.com', 'active', true),
    ('Quinn Garcia', 'quinn.garcia@example.com', 'active', true),
    ('Rachel Martinez', 'rachel.martinez@example.com', 'suspended', false),
    ('Samuel Rodriguez', 'samuel.rodriguez@example.com', 'active', true),
    ('Tara Lewis', 'tara.lewis@example.com', 'active', true);

-- Insert sample profiles for some users
INSERT INTO profiles (user_id, bio, avatar_url, website, location) VALUES
    (1, 'Software engineer passionate about web development and open source.', 'https://avatars.example.com/alice', 'https://alicesmith.dev', 'San Francisco, CA'),
    (2, 'Product manager with 10+ years experience in tech startups.', 'https://avatars.example.com/bob', 'https://bobjohnson.com', 'New York, NY'),
    (3, 'UX/UI designer creating beautiful and functional user experiences.', 'https://avatars.example.com/carol', 'https://caroldesigns.com', 'Austin, TX'),
    (5, 'Data scientist specializing in machine learning and AI.', 'https://avatars.example.com/eva', 'https://evadavis.ai', 'Boston, MA'),
    (6, 'DevOps engineer focused on cloud infrastructure and automation.', 'https://avatars.example.com/frank', NULL, 'Seattle, WA'),
    (8, 'Full-stack developer and tech blogger.', 'https://avatars.example.com/henry', 'https://henrymoore.blog', 'Chicago, IL'),
    (9, 'Mobile app developer with expertise in React Native.', 'https://avatars.example.com/isabella', 'https://isabellaapps.com', 'Los Angeles, CA'),
    (10, 'Security engineer and ethical hacker.', 'https://avatars.example.com/jack', 'https://jacksec.net', 'Denver, CO'),
    (13, 'Frontend developer and accessibility advocate.', 'https://avatars.example.com/mia', 'https://miawhite.dev', 'Portland, OR'),
    (14, 'Backend engineer specializing in microservices.', 'https://avatars.example.com/noah', NULL, 'Miami, FL'),
    (15, 'Technical writer and documentation specialist.', 'https://avatars.example.com/olivia', 'https://oliviawrites.tech', 'Atlanta, GA'),
    (16, 'Cloud architect with AWS and Azure expertise.', 'https://avatars.example.com/paul', 'https://paulcloud.com', 'Phoenix, AZ'),
    (19, 'QA engineer and test automation specialist.', 'https://avatars.example.com/samuel', NULL, 'Nashville, TN'),
    (20, 'Database administrator and performance tuning expert.', 'https://avatars.example.com/tara', 'https://taradb.com', 'Charlotte, NC');

-- Add some users with specific patterns for testing search functionality
INSERT INTO users (name, email, status, active) VALUES
    ('Alice Johnson', 'alice.j@example.com', 'active', true),
    ('Alice Brown', 'alice.brown@example.com', 'active', true),
    ('Alex Smith', 'alex.smith@example.com', 'active', true);

-- Verify data insertion
-- These comments show expected counts after insertion:
-- Total users: 23
-- Active users: 19  
-- Inactive users: 2
-- Suspended users: 3
-- Users with profiles: 14