-- Rollback 061: restore the inline UNIQUE constraint on systems(system_key)

ALTER TABLE systems ADD CONSTRAINT systems_system_key_key UNIQUE (system_key);
