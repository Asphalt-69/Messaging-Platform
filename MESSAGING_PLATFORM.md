# Advanced Messaging Platform

A modern, feature-rich messaging application built with Next.js, React, and Supabase. Supports 1-to-1 direct messages, group chats, voice messages, file attachments, emojis, message reactions, and replies.

## Features

### Core Messaging
- **1-to-1 Direct Messages**: Private conversations between two users
- **Group Chats**: Create and manage group conversations with multiple participants
- **Real-time Updates**: Instant message delivery and notifications
- **Message History**: Full conversation history with timestamps

### Advanced Features
- **Voice Messages**: Record and send audio messages up to 60 seconds
- **File Attachments**: Share images, documents, and other files
- **Emoji Picker**: Quick emoji insertion with 12 common emojis
- **Message Reactions**: React to messages with emojis (👍, ❤️, 😂, 😮, 😢, 🔥)
- **Message Replies**: Quote and reply to specific messages in conversations
- **Read Receipts**: Track message read status
- **Typing Indicators**: See when others are typing

### User Interface
- **Responsive Design**: Mobile-first design that works on all devices
- **Chat List Sidebar**: Search and filter conversations
- **Message View**: Clean, organized message display
- **User Avatars**: Profile pictures and status indicators
- **Search**: Full-text search across conversations and messages

## Technology Stack

- **Frontend**: Next.js 16, React 19, TypeScript
- **Styling**: Tailwind CSS, shadcn/ui components
- **Backend**: Next.js API Routes
- **Database**: Supabase (PostgreSQL)
- **Storage**: Supabase Storage (for files and voice messages)
- **Authentication**: Supabase Auth
- **Icons**: Lucide React
- **Date Formatting**: date-fns

## Setup Instructions

### Prerequisites
- Node.js 18+ and npm/pnpm
- Supabase account (free tier available at supabase.com)

### 1. Clone and Install

```bash
git clone <your-repo-url>
cd messaging-platform
npm install
# or
pnpm install
```

### 2. Setup Supabase

1. Create a new project on [Supabase](https://supabase.com)
2. Go to Settings → Database and note your Project URL and Anon Key
3. Run the database migration:
   - Execute `scripts/setup-messaging-db.sql` in your Supabase SQL editor
   - Execute `scripts/seed-data.sql` to populate sample data (optional)

### 3. Environment Variables

Create a `.env.local` file in the project root:

```env
NEXT_PUBLIC_SUPABASE_URL=your_supabase_url
NEXT_PUBLIC_SUPABASE_ANON_KEY=your_anon_key
```

### 4. Configure Storage

In Supabase, create two storage buckets:

1. `message-attachments` - For file uploads (public)
   - Enable "File size limit": 100 MB
   - Add policy to allow authenticated users to upload

2. `voice-messages` - For voice recordings (public)
   - Enable "File size limit": 50 MB
   - Add policy to allow authenticated users to upload

Example policy for message-attachments:
```sql
CREATE POLICY "Allow authenticated uploads" 
ON storage.objects 
FOR INSERT TO authenticated
WITH CHECK (bucket_id = 'message-attachments');
```

### 5. Enable Row Level Security (RLS)

Apply RLS policies to your tables:

```sql
-- Enable RLS on all tables
ALTER TABLE profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE conversations ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE message_reactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE message_attachments ENABLE ROW LEVEL SECURITY;

-- Example RLS policy for messages
CREATE POLICY "Users can view messages in their conversations"
ON messages FOR SELECT
USING (
  EXISTS (
    SELECT 1 FROM conversations
    WHERE conversations.id = messages.conversation_id
    AND (conversations.user1_id = auth.uid() OR conversations.user2_id = auth.uid())
  ) OR
  EXISTS (
    SELECT 1 FROM conversation_members
    WHERE conversation_id = messages.conversation_id
    AND user_id = auth.uid()
  )
);
```

### 6. Run the Development Server

```bash
npm run dev
# or
pnpm dev
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the application.

## Database Schema

### Tables
- **profiles**: User information (id, name, avatar, created_at)
- **conversations**: Chat conversations (id, name, is_group, user1_id, user2_id, etc.)
- **conversation_members**: Group chat members
- **messages**: Chat messages (id, conversation_id, sender_id, content, created_at, etc.)
- **message_reactions**: Emoji reactions on messages
- **message_attachments**: File attachments metadata
- **read_receipts**: Message read status

## API Endpoints

### Chats
- `GET /api/chats` - Get all conversations
- `GET /api/chats/[id]` - Get chat details
- `GET /api/chats/[id]/messages` - Get messages in a conversation

### Messages
- `POST /api/messages` - Send a message (with optional attachments/voice)
- `POST /api/messages/[id]/reactions` - Add reaction to a message

## Component Structure

```
components/
├── chat-list.tsx        # Conversation list sidebar
├── chat-view.tsx        # Main chat view
├── message-item.tsx     # Individual message display
├── message-input.tsx    # Message input with advanced features
└── ui/                  # UI components (Button, Input, Textarea)

app/
├── page.tsx             # Main layout
├── layout.tsx           # Root layout
└── api/
    ├── chats/           # Chat-related endpoints
    └── messages/        # Message-related endpoints
```

## Features in Detail

### Voice Messages
- Click the microphone button when the text field is empty
- Records audio until you click stop
- Automatically sends with duration metadata
- Displays as playable audio player in chat

### File Attachments
- Click the paperclip icon to attach files
- Supports images, documents, and any file type
- Multiple files can be attached to a single message
- Images display inline, other files show as downloadable

### Emoji Reactions
- Hover over a message and click the smile icon
- Choose from 12 quick-reaction emojis
- Or use the emoji picker in the message input
- See reaction count and who reacted

### Message Replies
- Hover over a message and click the reply icon
- Original message appears in a quoted format
- Reply preview shows before sending
- Full conversation context preserved

## Development

### Build for Production
```bash
npm run build
npm start
```

### Linting
```bash
npm run lint
```

## Deployment

### Deploy to Vercel (Recommended)
1. Push your code to GitHub
2. Connect repository to Vercel
3. Add environment variables in Vercel dashboard
4. Deploy!

## Future Enhancements

- [ ] Real-time updates with Supabase subscriptions
- [ ] Typing indicators
- [ ] Online status
- [ ] Message search with full-text indexing
- [ ] Video calls with WebRTC
- [ ] Message editing and deletion
- [ ] User presence and activity status
- [ ] Custom emojis and stickers
- [ ] Message forwarding
- [ ] Pin important messages
- [ ] Message reactions with custom emojis

## License

MIT

## Support

For issues or questions, please create an issue in the GitHub repository.
