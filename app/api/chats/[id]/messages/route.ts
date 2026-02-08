import { createClient } from '@supabase/supabase-js'
import { NextRequest, NextResponse } from 'next/server'

const supabase = createClient(
  process.env.NEXT_PUBLIC_SUPABASE_URL!,
  process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!
)

export async function GET(
  request: NextRequest,
  { params }: { params: { id: string } }
) {
  try {
    const { id } = params

    // Get current user
    const {
      data: { user },
    } = await supabase.auth.getUser()
    if (!user) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    // Fetch messages for the conversation
    const { data, error } = await supabase
      .from('messages')
      .select(
        `
        *,
        sender:profiles(id, name, avatar),
        reactions:message_reactions(*),
        attachments:message_attachments(*)
      `
      )
      .eq('conversation_id', id)
      .order('created_at', { ascending: true })

    if (error) throw error

    // Map messages to the expected format
    const messages = data.map((msg: any) => ({
      id: msg.id,
      content: msg.content,
      senderId: msg.sender_id,
      senderName: msg.sender?.name || 'Unknown',
      senderAvatar: msg.sender?.avatar,
      timestamp: new Date(msg.created_at),
      attachments: msg.attachments?.map((att: any) => ({
        id: att.id,
        url: att.url,
        type: att.type,
        name: att.name,
      })),
      reactions: msg.reactions?.reduce((acc: any[], r: any) => {
        const existing = acc.find((item) => item.emoji === r.emoji)
        if (existing) {
          existing.count++
          existing.users.push(r.user_id)
        } else {
          acc.push({ emoji: r.emoji, count: 1, users: [r.user_id] })
        }
        return acc
      }, []),
      isVoiceMessage: msg.is_voice_message,
      voiceDuration: msg.voice_duration,
      replyTo: msg.reply_to_id
        ? {
            id: msg.reply_to_id,
            content: msg.reply_to_content,
            senderName: msg.reply_to_sender_name,
          }
        : undefined,
      isOwn: msg.sender_id === user.id,
    }))

    return NextResponse.json(messages)
  } catch (error) {
    console.error('Error fetching messages:', error)
    return NextResponse.json(
      { error: 'Failed to fetch messages' },
      { status: 500 }
    )
  }
}
