-- Vocabulary drills (generated from vocab): headword -> meaning, meaning -> headword.
ALTER TABLE question_types
  MODIFY format ENUM('mcq','vocab_meaning','vocab_word') NOT NULL;
