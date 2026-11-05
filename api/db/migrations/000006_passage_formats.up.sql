-- Passage-based questions (大問2 長文の語句空所補充, 大問3 長文の内容一致選択).
-- Each passage question is stored as an ordinary question; its payload
-- carries the passage, and its stable key is "<passage key>-<question id>".
ALTER TABLE question_types
  MODIFY format ENUM('mcq','vocab_meaning','vocab_word','passage_blank','passage_question') NOT NULL;
