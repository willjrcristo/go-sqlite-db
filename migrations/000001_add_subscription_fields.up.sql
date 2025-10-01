-- migrations/000001_add_subscription_fields.up.sql
ALTER TABLE usuarios ADD COLUMN stripe_customer_id TEXT;
ALTER TABLE usuarios ADD COLUMN stripe_subscription_id TEXT;
ALTER TABLE usuarios ADD COLUMN subscription_status TEXT DEFAULT 'inactive';
ALTER TABLE usuarios ADD COLUMN subscription_current_period_end DATETIME;