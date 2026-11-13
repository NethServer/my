-- Rollback Migration 059: Drop the services added by operators to support sessions

DROP TABLE IF EXISTS support_system_services;
