import { createClient } from '@supabase/supabase-js'
import { NextRequest, NextResponse } from 'next/server'

const supabase = createClient(
  process.env.NEXT_PUBLIC_SUPABASE_URL!,
  process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!
)

export async function POST(request: NextRequest) {
  try {
    // Get current user
    const {
      data: { user },
    } = await supabase.auth.getUser()
    if (!user) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const formData = await request.formData()
    const content = formData.get('content') as string
    const chatId = formData.get('chatId') as string
    const replyToId = formData.get('replyToId') as string | null
    const isVoiceMessage = formData.get('isVoiceMessage') === 'true'
    const duration = formData.get('duration')
      ? parseInt(formData.get('duration') as string)
      : null

    // Create message
    const { data: messageData, error: messageError } = await supabase
      .from('messages')
      .insert({
        conversation_id: chatId,
        sender_id: user.id,
        content: content || null,
        is_voice_message: isVoiceMessage,
        voice_duration: duration,
        reply_to_id: replyToId || null,
      })
      .select()
      .single()

    if (messageError) throw messageError

    // Handle file attachments
    const attachmentFiles = formData.getAll('attachments') as File[]
    if (attachmentFiles.length > 0) {
      for (const file of attachmentFiles) {
        const fileExtension = file.name.split('.').pop()
        const fileName = `${user.id}/${chatId}/${Date.now()}-${Math.random()}.${fileExtension}`

        // Upload to storage
        const { error: uploadError } = await supabase.storage
          .from('message-attachments')
          .upload(fileName, file)

        if (uploadError) throw uploadError

        // Get public URL
        const { data: publicUrl } = supabase.storage
          .from('message-attachments')
          .getPublicUrl(fileName)

        // Store attachment metadata
        await supabase.from('message_attachments').insert({
          message_id: messageData.id,
          url: publicUrl.publicUrl,
          type: file.type.startsWith('image') ? 'image' : 'file',
          name: file.name,
        })
      }
    }

    // Handle voice message
    if (isVoiceMessage) {
      const audioBlob = formData.get('audioBlob') as Blob
      if (audioBlob) {
        const fileName = `${user.id}/${chatId}/${Date.now()}.webm`

        const { error: uploadError } = await supabase.storage
          .from('voice-messages')
          .upload(fileName, audioBlob)

        if (uploadError) throw uploadError

        const { data: publicUrl } = supabase.storage
          .from('voice-messages')
          .getPublicUrl(fileName)

        await supabase
          .from('messages')
          .update({ voice_url: publicUrl.publicUrl })
          .eq('id', messageData.id)
      }
    }

    return NextResponse.json(messageData)
  } catch (error) {
    console.error('Error sending message:', error)
    return NextResponse.json(
      { error: 'Failed to send message' },
      { status: 500 }
    )
  }
}
