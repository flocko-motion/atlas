-- Add title field to nodes — short subject line for display and LLM triage.
ALTER TABLE nodes ADD COLUMN title TEXT;
