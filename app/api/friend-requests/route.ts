import { createClient } from '@supabase/supabase-js'
import { NextRequest, NextResponse } from 'next/server'

const supabase = createClient(
  process.env.NEXT_PUBLIC_SUPABASE_URL || '',
  process.env.SUPABASE_SERVICE_ROLE_KEY || ''
)

export async function GET(req: NextRequest) {
  try {
    const userId = req.nextUrl.searchParams.get('userId')

    if (!userId) {
      return NextResponse.json(
        { error: 'User ID is required' },
        { status: 400 }
      )
    }

    // Get pending friend requests
    const { data, error } = await supabase
      .from('friend_requests')
      .select(
        `
        id,
        sender_id,
        sender:users!sender_id (id, name, username, avatar_url),
        status,
        created_at
      `
      )
      .eq('recipient_id', userId)
      .eq('status', 'pending')
      .order('created_at', { ascending: false })

    if (error) throw error

    return NextResponse.json(data)
  } catch (error) {
    return NextResponse.json(
      { error: 'Failed to fetch friend requests' },
      { status: 500 }
    )
  }
}

export async function POST(req: NextRequest) {
  try {
    const { recipient_username } = await req.json()

    if (!recipient_username) {
      return NextResponse.json(
        { error: 'Recipient username is required' },
        { status: 400 }
      )
    }

    // Find recipient by username
    const { data: recipient, error: recipientError } = await supabase
      .from('users')
      .select('id')
      .eq('username', recipient_username)
      .single()

    if (recipientError || !recipient) {
      return NextResponse.json(
        { error: 'User not found' },
        { status: 404 }
      )
    }

    // Check if request already exists
    const { data: existing } = await supabase
      .from('friend_requests')
      .select('id')
      .eq('sender_id', 'current_user_id')
      .eq('recipient_id', recipient.id)
      .in('status', ['pending', 'accepted'])
      .single()

    if (existing) {
      return NextResponse.json(
        { error: 'Friend request already exists' },
        { status: 400 }
      )
    }

    // Create friend request
    const { data, error } = await supabase
      .from('friend_requests')
      .insert([
        {
          sender_id: 'current_user_id',
          recipient_id: recipient.id,
          status: 'pending',
        },
      ])
      .select()
      .single()

    if (error) throw error

    return NextResponse.json(data, { status: 201 })
  } catch (error) {
    return NextResponse.json(
      { error: 'Failed to send friend request' },
      { status: 500 }
    )
  }
}

export async function PUT(req: NextRequest) {
  try {
    const { requestId, action } = await req.json()

    if (!requestId || !['accept', 'reject'].includes(action)) {
      return NextResponse.json(
        { error: 'Request ID and action are required' },
        { status: 400 }
      )
    }

    // Update friend request
    const { data, error } = await supabase
      .from('friend_requests')
      .update({
        status: action === 'accept' ? 'accepted' : 'rejected',
        updated_at: new Date(),
      })
      .eq('id', requestId)
      .select()
      .single()

    if (error) throw error

    // If accepted, add to friends table
    if (action === 'accept') {
      const { data: request } = await supabase
        .from('friend_requests')
        .select('sender_id, recipient_id')
        .eq('id', requestId)
        .single()

      if (request) {
        await supabase.from('friends').insert([
          {
            user_id: request.sender_id,
            friend_id: request.recipient_id,
          },
          {
            user_id: request.recipient_id,
            friend_id: request.sender_id,
          },
        ])
      }
    }

    return NextResponse.json(data)
  } catch (error) {
    return NextResponse.json(
      { error: 'Failed to update friend request' },
      { status: 500 }
    )
  }
}
