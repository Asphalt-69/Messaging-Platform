-- Seed sample data for testing the messaging platform

-- Insert sample users/profiles
INSERT INTO profiles (id, name, avatar, created_at) VALUES
  ('user-1', 'Alice Johnson', 'A', NOW()),
  ('user-2', 'Bob Smith', 'B', NOW()),
  ('user-3', 'Carol Davis', 'C', NOW()),
  ('user-4', 'David Wilson', 'D', NOW())
ON CONFLICT (id) DO NOTHING;

-- Insert sample 1-to-1 conversations
INSERT INTO conversations (id, name, is_group, user1_id, user2_id, created_at, updated_at) VALUES
  ('conv-1', 'Alice Johnson', false, 'user-1', 'user-2', NOW(), NOW()),
  ('conv-2', 'Carol Davis', false, 'user-1', 'user-3', NOW(), NOW()),
  ('conv-3', 'David Wilson', false, 'user-1', 'user-4', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Insert sample group conversation
INSERT INTO conversations (id, name, is_group, member_count, created_at, updated_at) VALUES
  ('conv-group-1', 'Development Team', true, 4, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Insert sample messages for conv-1
INSERT INTO messages (id, conversation_id, sender_id, content, created_at) VALUES
  ('msg-1', 'conv-1', 'user-2', 'Hey Alice! How are you?', NOW() - interval '2 hours'),
  ('msg-2', 'conv-1', 'user-1', 'Hi Bob! I''m doing great, thanks for asking!', NOW() - interval '1 hour 55 minutes'),
  ('msg-3', 'conv-1', 'user-2', 'That''s awesome! Want to grab coffee later?', NOW() - interval '1 hour 50 minutes'),
  ('msg-4', 'conv-1', 'user-1', 'Sure! How about 3 PM at the usual place?', NOW() - interval '1 hour 45 minutes'),
  ('msg-5', 'conv-1', 'user-2', 'Perfect! See you then 😊', NOW() - interval '1 hour 40 minutes')
ON CONFLICT (id) DO NOTHING;

-- Insert sample messages for conv-2
INSERT INTO messages (id, conversation_id, sender_id, content, created_at) VALUES
  ('msg-6', 'conv-2', 'user-3', 'Did you see the new design mockups?', NOW() - interval '30 minutes'),
  ('msg-7', 'conv-2', 'user-1', 'Yes! They look fantastic. I love the new layout.', NOW() - interval '25 minutes'),
  ('msg-8', 'conv-2', 'user-3', 'Me too! Ready to present them to the team?', NOW() - interval '20 minutes')
ON CONFLICT (id) DO NOTHING;

-- Insert sample messages for conv-3 with reply
INSERT INTO messages (id, conversation_id, sender_id, content, created_at) VALUES
  ('msg-9', 'conv-3', 'user-4', 'The project deadline is next Friday', NOW() - interval '15 minutes'),
  ('msg-10', 'conv-3', 'user-1', 'Got it, I''ll make sure everything is ready', NOW() - interval '10 minutes'),
  ('msg-11', 'conv-3', 'user-4', 'Thanks! Really appreciate your help', NOW() - interval '5 minutes')
ON CONFLICT (id) DO NOTHING;

-- Insert sample group messages
INSERT INTO messages (id, conversation_id, sender_id, content, created_at) VALUES
  ('msg-12', 'conv-group-1', 'user-2', 'Morning team! Daily standup in 10 minutes', NOW() - interval '15 minutes'),
  ('msg-13', 'conv-group-1', 'user-3', 'Ready! Just finishing up some code reviews', NOW() - interval '12 minutes'),
  ('msg-14', 'conv-group-1', 'user-4', 'On my way, just wrapping up a call', NOW() - interval '8 minutes'),
  ('msg-15', 'conv-group-1', 'user-1', 'Let''s get started!', NOW() - interval '5 minutes')
ON CONFLICT (id) DO NOTHING;

-- Add some reactions to messages
INSERT INTO message_reactions (id, message_id, user_id, emoji, created_at) VALUES
  (gen_random_uuid(), 'msg-5', 'user-1', '👍', NOW()),
  (gen_random_uuid(), 'msg-7', 'user-3', '❤️', NOW()),
  (gen_random_uuid(), 'msg-12', 'user-1', '👏', NOW()),
  (gen_random_uuid(), 'msg-12', 'user-3', '👏', NOW())
ON CONFLICT DO NOTHING;

-- Update conversation last messages
UPDATE conversations SET 
  last_message = 'Perfect! See you then 😊',
  updated_at = NOW()
WHERE id = 'conv-1';

UPDATE conversations SET 
  last_message = 'Me too! Ready to present them to the team?',
  updated_at = NOW()
WHERE id = 'conv-2';

UPDATE conversations SET 
  last_message = 'Thanks! Really appreciate your help',
  updated_at = NOW()
WHERE id = 'conv-3';

UPDATE conversations SET 
  last_message = 'Let''s get started!',
  updated_at = NOW()
WHERE id = 'conv-group-1';
