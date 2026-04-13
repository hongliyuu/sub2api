-- 103_make_payment_orders_out_trade_no_unique.sql
--
-- Normalizes historical blank or duplicated out_trade_no values before
-- enforcing uniqueness. Older upgrades backfilled existing rows with '',
-- which breaks webhook verification and order lookup paths that assume
-- out_trade_no uniquely identifies a payment order.
--
-- Idempotent: safe to run multiple times.

UPDATE payment_orders
   SET out_trade_no = '__migration_103_legacy__' || CAST(id AS TEXT)
 WHERE out_trade_no = '';

WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY out_trade_no ORDER BY id) AS rn
      FROM payment_orders
     WHERE out_trade_no <> ''
)
UPDATE payment_orders
   SET out_trade_no = '__migration_103_dup__' || CAST(id AS TEXT)
 WHERE id IN (SELECT id FROM ranked WHERE rn > 1);

DROP INDEX IF EXISTS paymentorder_out_trade_no;
CREATE UNIQUE INDEX IF NOT EXISTS paymentorder_out_trade_no ON payment_orders (out_trade_no);
