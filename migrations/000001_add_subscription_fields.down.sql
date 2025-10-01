-- migrations/000001_add_subscription_fields.down.sql
ALTER TABLE usuarios DROP COLUMN stripe_customer_id;
ALTER TABLE usuarios DROP COLUMN stripe_subscription_id;
ALTER TABLE usuarios DROP COLUMN subscription_status;
ALTER TABLE usuarios DROP COLUMN subscription_current_period_end;