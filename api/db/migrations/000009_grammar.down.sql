DELETE FROM attempts WHERE mode = 'grammar';
ALTER TABLE attempts
  DROP COLUMN grammar_unit,
  MODIFY mode ENUM('practice','daily','placement') NOT NULL;
DROP TABLE grammar_units;
