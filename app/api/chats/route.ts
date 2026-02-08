import { createClient } from '@supabase/supabase-js'
import { NextRequest, NextResponse } from 'next/server'

const supabase = createClient(
  process.env.NEXT_PUBLIC_SUPABASE_URL!,
  process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!
)

export async function GET(request: NextRequest) {
  try {
    // Get current user from auth
    const {
      data: { user },
    } = await supabase.auth.getUser()
    if (!user) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    // Fetch all conversations for the user
    const { data, error } = await supabase
      .from('conversations')
      .select('*')
      .or(`user1_id.eq.${user.id},user2_id.eq.${user.id}`)
      .order('updated_at', { ascending: false })

    if (error) throw error

    // Map conversations to chat format
    const chats = data.map((conv: any) => ({
      id: conv.id,
      name: conv.name,
      avatar: conv.avatar,
      isGroup: conv.is_group,
      members: conv.member_count,
      lastMessage: conv.last_message,
      lastMessageTime: new Date(conv.updated_at).toLocaleTimeString(),
      unreadCount: conv.unread_count || 0,
    }))

    return NextResponse.json(chats)
  } catch (error) {
    console.error('Error fetching chats:', error)
    return NextResponse.json(
      { error: 'Failed to fetch chats' },
      { status: 500 }
    )
  }
}
