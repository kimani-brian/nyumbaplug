-- Email OTP verification for customer (tenant) and agent (landlord) registration.
ALTER TABLE users
  ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN otp_code TEXT,
  ADD COLUMN otp_expires_at TIMESTAMPTZ,
  ADD COLUMN otp_sent_at TIMESTAMPTZ;

-- Existing accounts predate email verification; mark them verified so nothing
-- changes for current users. New registrations default to FALSE and must enter
-- the OTP sent at signup.
UPDATE users SET email_verified = TRUE;
