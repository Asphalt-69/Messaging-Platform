'use client'

import { useState, useEffect, useRef } from 'react'
import { ChevronLeft, Phone, Video, MoreVertical } from 'lucide-react'
import { Button } from '@/components/ui/button'
import MessageItem from './message-item'
import MessageInput from './message-input'

interface Message {
  id: string
  content?: string
  senderId: string
  senderName: string
  senderAvatar?: string
  timestamp: Date
  attachments?: Array<{
    id: string
    url: string
    type: 'image' | 'file' | 'voice'
    name?: string
  }>
  reactions?: Array<{
    emoji: string
    count: number
    users: string[]
  }>
  isVoiceMessage?: boolean
  voiceDuration?: number
  replyTo?: {
    id: string
    content: string
    senderName: string
  }
  isOwn: boolean
}

interface Chat {
  id: string
  name: string
  avatar?: string
  isGroup?: boolean
  members?: number
}

interface ChatViewProps {
  chatId?: string
  onBackClick?: () => void
}

export default function ChatView({ chatId, onBackClick }: ChatViewProps) {
  const [chat, setChat] = useState<Chat | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [replyingTo, setReplyingTo] = useState<Message | null>(null)
  const [loading, setLoading] = useState(true)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (chatId) {
      fetchChat()
      fetchMessages()
    }
  }, [chatId])

  useEffect(() => {
    scrollToBottom()
  }, [messages])

  const fetchChat = async () => {
    try {
      const response = await fetch(`/api/chats/${chatId}`)
      const data = await response.json()
      setChat(data)
    } catch (error) {
      console.log('[v0] Error fetching chat:', error)
    }
  }

  const fetchMessages = async () => {
    try {
      setLoading(true)
      const response = await fetch(`/api/chats/${chatId}/messages`)
      const data = await response.json()
      setMessages(data)
    } catch (error) {
      console.log('[v0] Error fetching messages:', error)
    } finally {
      setLoading(false)
    }
  }

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  const handleSendMessage = async (content: string, attachments: File[]) => {
    try {
      const formData = new FormData()
      formData.append('content', content)
      formData.append('chatId', chatId!)
      if (replyingTo) {
        formData.append('replyToId', replyingTo.id)
      }

      attachments.forEach((file) => {
        formData.append('attachments', file)
      })

      const response = await fetch('/api/messages', {
        method: 'POST',
        body: formData,
      })

      if (response.ok) {
        const newMessage = await response.json()
        setMessages((prev) => [...prev, newMessage])
        setReplyingTo(null)
      }
    } catch (error) {
      console.log('[v0] Error sending message:', error)
    }
  }

  const handleSendVoiceMessage = async (audioBlob: Blob, duration: number) => {
    try {
      const formData = new FormData()
      formData.append('audioBlob', audioBlob)
      formData.append('duration', duration.toString())
      formData.append('chatId', chatId!)
      formData.append('isVoiceMessage', 'true')

      const response = await fetch('/api/messages', {
        method: 'POST',
        body: formData,
      })

      if (response.ok) {
        const newMessage = await response.json()
        setMessages((prev) => [...prev, newMessage])
      }
    } catch (error) {
      console.log('[v0] Error sending voice message:', error)
    }
  }

  const handleReply = (message: Message) => {
    setReplyingTo(message)
  }

  const handleAddReaction = async (messageId: string, emoji: string) => {
    try {
      const response = await fetch(`/api/messages/${messageId}/reactions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ emoji }),
      })

      if (response.ok) {
        // Update local state
        setMessages((prev) =>
          prev.map((msg) => {
            if (msg.id === messageId) {
              const existingReaction = msg.reactions?.find(
                (r) => r.emoji === emoji
              )
              if (existingReaction) {
                return {
                  ...msg,
                  reactions: msg.reactions?.map((r) =>
                    r.emoji === emoji ? { ...r, count: r.count + 1 } : r
                  ),
                }
              } else {
                return {
                  ...msg,
                  reactions: [
                    ...(msg.reactions || []),
                    { emoji, count: 1, users: [] },
                  ],
                }
              }
            }
            return msg
          })
        )
      }
    } catch (error) {
      console.log('[v0] Error adding reaction:', error)
    }
  }

  if (!chatId) {
    return (
      <div className="h-full flex items-center justify-center bg-background text-muted-foreground">
        <p>Select a conversation to start messaging</p>
      </div>
    )
  }

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center bg-background">
        <p>Loading chat...</p>
      </div>
    )
  }

  return (
    <div className="h-full flex flex-col bg-background">
      {/* Header */}
      {chat && (
        <div className="border-b border-border p-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            {onBackClick && (
              <Button
                size="icon"
                variant="ghost"
                onClick={onBackClick}
                className="md:hidden"
              >
                <ChevronLeft className="h-5 w-5" />
              </Button>
            )}
            <div className="w-12 h-12 rounded-full bg-accent flex items-center justify-center text-accent-foreground font-semibold flex-shrink-0">
              {chat.avatar || chat.name[0]?.toUpperCase()}
            </div>
            <div>
              <h2 className="font-semibold">{chat.name}</h2>
              {chat.isGroup && chat.members && (
                <p className="text-xs text-muted-foreground">
                  {chat.members} members
                </p>
              )}
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Button size="icon" variant="ghost">
              <Phone className="h-5 w-5" />
            </Button>
            <Button size="icon" variant="ghost">
              <Video className="h-5 w-5" />
            </Button>
            <Button size="icon" variant="ghost">
              <MoreVertical className="h-5 w-5" />
            </Button>
          </div>
        </div>
      )}

      {/* Messages */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {messages.length === 0 ? (
          <div className="flex items-center justify-center h-full text-muted-foreground">
            <p>No messages yet. Start the conversation!</p>
          </div>
        ) : (
          messages.map((message) => (
            <MessageItem
              key={message.id}
              message={message}
              onReply={handleReply}
              onAddReaction={handleAddReaction}
            />
          ))
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* Input */}
      <MessageInput
        onSendMessage={handleSendMessage}
        onSendVoiceMessage={handleSendVoiceMessage}
        replyingTo={
          replyingTo
            ? {
                id: replyingTo.id,
                content: replyingTo.content || '',
                senderName: replyingTo.senderName,
              }
            : undefined
        }
        onCancelReply={() => setReplyingTo(null)}
      />
    </div>
  )
}
