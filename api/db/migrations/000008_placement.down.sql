ALTER TABLE learner_profiles DROP COLUMN placement_stage;
DELETE FROM attempts WHERE mode = 'placement';
ALTER TABLE attempts
  MODIFY mode ENUM('practice','daily') NOT NULL;
