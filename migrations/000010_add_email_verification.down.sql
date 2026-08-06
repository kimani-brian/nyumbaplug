ALTER TABLE users
  DROP COLUMN email_verified,
  DROP COLUMN otp_code,
  DROP COLUMN otp_expires_at,
  DROP COLUMN otp_sent_at;