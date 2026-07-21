-- Rollback migration 038: restore default autoanalyze behaviour
ALTER TABLE systems      RESET (autovacuum_analyze_scale_factor);
ALTER TABLE applications RESET (autovacuum_analyze_scale_factor);
ALTER TABLE distributors RESET (autovacuum_analyze_scale_factor, autovacuum_analyze_threshold);
