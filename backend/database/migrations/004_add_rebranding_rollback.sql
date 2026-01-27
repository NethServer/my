-- Rollback: Remove rebranding tables

DROP TABLE IF EXISTS rebranding_assets CASCADE;
DROP TABLE IF EXISTS rebranding_enabled CASCADE;
DROP TABLE IF EXISTS rebrandable_products CASCADE;
