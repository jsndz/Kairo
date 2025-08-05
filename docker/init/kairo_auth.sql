-- Create custom user with password
CREATE USER admin WITH PASSWORD 'admin';

-- Grant privileges on DB
GRANT ALL PRIVILEGES ON DATABASE kairo_auth TO kairo_user;
