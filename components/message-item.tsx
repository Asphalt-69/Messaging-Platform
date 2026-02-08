'use client'

import { useState } from 'react'
import { MessageCircle, Smile, Download, Check, CheckCheck, Clock, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { formatDistanceToNow } from 'date-fns'

interface Attachment {
  id: string
  url: string
  type: 'image' | 'file' | 'voice'
  name?: string
}

interface Reaction {
  emoji: string
  count: number
  users: string[]
}

type MessageStatus = 'sending' | 'sent' | 'delivered' | 'read' | 'unsent'

interface Message {
  id: string
  content?: string
  senderId: string
  senderName: string
  senderAvatar?: string
  senderImage?: string
  timestamp: Date
  attachments?: Attachment[]
  reactions?: Reaction[]
  isVoiceMessage?: boolean
  voiceDuration?: number
  replyTo?: {
    id: string
    content: string
    senderName: string
  }
  isOwn: boolean
  status?: MessageStatus
}

interface MessageItemProps {
  message: Message
  onReply: (message: Message) => void
  onAddReaction: (messageId: string, emoji: string) => void
}

export default function MessageItem({
  message,
  onReply,
  onAddReaction,
}: MessageItemProps) {
  const [showReactions, setShowReactions] = useState(false)
  const [showDelete, setShowDelete] = useState(false)

  const commonEmojis = ['👍', '❤️', '😂', '😮', '😢', '🔥']

  const getStatusIcon = (status?: MessageStatus) => {
    switch (status) {
      case 'sending':
        return <Clock className="h-3 w-3" title="Sending..." />
      case 'sent':
        return <Check className="h-3 w-3" title="Sent" />
      case 'delivered':
        return <CheckCheck className="h-3 w-3" title="Delivered" />
      case 'read':
        return <CheckCheck className="h-3 w-3 text-blue-500" title="Read" />
      case 'unsent':
        return <Trash2 className="h-3 w-3 text-destructive" title="Unsent" />
      default:
        return null
    }
  }

  return (
    <div
      className={`flex gap-3 mb-4 ${
        message.isOwn ? 'flex-row-reverse' : 'flex-row'
      }`}
    >
      {/* Avatar */}
      {!message.isOwn && (
        <div className="w-8 h-8 rounded-full bg-accent flex-shrink-0 flex items-center justify-center text-accent-foreground text-xs font-bold overflow-hidden">
          {message.senderImage ? (
            <img src={message.senderImage} alt={message.senderName} className="w-full h-full object-cover" />
          ) : (
            message.senderAvatar || message.senderName[0]?.toUpperCase()
          )}
        </div>
      )}

      <div className={`flex flex-col gap-1 max-w-xs ${message.isOwn ? 'items-end' : 'items-start'}`}>
        {/* Sender Info */}
        {!message.isOwn && (
          <span className="text-xs font-semibold text-foreground px-3">
            {message.senderName}
          </span>
        )}

        {/* Reply To */}
        {message.replyTo && (
          <div
            className={`px-3 py-2 rounded-lg border-l-2 border-accent ${
              message.isOwn
                ? 'bg-accent/10 border-accent'
                : 'bg-muted border-muted-foreground'
            } max-w-xs`}
          >
            <p className="text-xs font-semibold text-muted-foreground">
              {message.replyTo.senderName}
            </p>
            <p className="text-sm text-foreground line-clamp-2">
              {message.replyTo.content}
            </p>
          </div>
        )}

        {/* Message Bubble */}
        <div
          className={`rounded-2xl px-4 py-2 ${
            message.isOwn
              ? 'bg-accent text-accent-foreground rounded-br-none'
              : 'bg-muted text-foreground rounded-bl-none'
          } relative group`}
        >
          {/* Voice Message */}
          {message.isVoiceMessage && message.voiceDuration && (
            <div className="flex items-center gap-3">
              <button className="w-8 h-8 rounded-full bg-accent-foreground/20 flex items-center justify-center hover:bg-accent-foreground/30 transition-colors">
                ▶️
              </button>
              <div className="flex-1">
                <div className="h-1 bg-accent-foreground/30 rounded-full"></div>
              </div>
              <span className="text-xs font-medium">
                {Math.floor(message.voiceDuration)}s
              </span>
            </div>
          )}

          {/* Text Content */}
          {message.content && (
            <p className="text-sm break-words whitespace-pre-wrap">
              {message.content}
            </p>
          )}

          {/* Attachments */}
          {message.attachments && message.attachments.length > 0 && (
            <div className="mt-2 space-y-2">
              {message.attachments.map((attachment) => (
                <div
                  key={attachment.id}
                  className="rounded-lg bg-accent-foreground/10 p-2"
                >
                  {attachment.type === 'image' ? (
                    <img
                      src={attachment.url}
                      alt="Attachment"
                      className="max-w-xs rounded-lg"
                    />
                  ) : (
                    <div className="flex items-center gap-2">
                      <Download className="h-4 w-4" />
                      <span className="text-xs truncate">
                        {attachment.name || 'File'}
                      </span>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}

          {/* Timestamp and Status */}
          <div
            className={`text-xs mt-1 flex items-center gap-1 ${
              message.isOwn
                ? 'text-accent-foreground/70'
                : 'text-muted-foreground'
            }`}
          >
            <span>{formatDistanceToNow(message.timestamp, { addSuffix: true })}</span>
            {message.isOwn && getStatusIcon(message.status)}
          </div>
        </div>

        {/* Reactions */}
        {message.reactions && message.reactions.length > 0 && (
          <div className="flex flex-wrap gap-1 mt-1">
            {message.reactions.map((reaction) => (
              <button
                key={reaction.emoji}
                className="px-2 py-1 rounded-full bg-muted hover:bg-accent transition-colors text-xs font-medium"
                title={reaction.users.join(', ')}
              >
                {reaction.emoji} {reaction.count}
              </button>
            ))}
          </div>
        )}

        {/* Action Buttons */}
        <div
          className={`flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity mt-1 ${
            message.isOwn ? 'flex-row-reverse' : 'flex-row'
          }`}
        >
          <Button
            size="sm"
            variant="ghost"
            className="h-7 w-7 p-0"
            onClick={() => setShowReactions(!showReactions)}
          >
            <Smile className="h-4 w-4" />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            className="h-7 w-7 p-0"
            onClick={() => onReply(message)}
          >
            <MessageCircle className="h-4 w-4" />
          </Button>
          {message.isOwn && (
            <Button
              size="sm"
              variant="ghost"
              className="h-7 w-7 p-0"
              onClick={() => setShowDelete(!showDelete)}
            >
              <Trash2 className="h-4 w-4 text-destructive" />
            </Button>
          )}
        </div>

        {/* Delete Confirmation */}
        {showDelete && message.isOwn && (
          <div className="text-xs bg-destructive/10 border border-destructive rounded p-2 mt-1">
            <p className="text-destructive mb-1">Unsend message?</p>
            <div className="flex gap-1">
              <Button
                size="sm"
                variant="destructive"
                className="h-5 text-xs flex-1"
                onClick={() => {
                  // Handle unsend
                  setShowDelete(false)
                }}
              >
                Unsend
              </Button>
              <Button
                size="sm"
                variant="outline"
                className="h-5 text-xs flex-1"
                onClick={() => setShowDelete(false)}
              >
                Cancel
              </Button>
            </div>
          </div>
        )}

        {/* Emoji Picker */}
        {showReactions && (
          <div className="flex gap-1 mt-1 bg-background border border-border rounded-lg p-2">
            {commonEmojis.map((emoji) => (
              <button
                key={emoji}
                className="p-1 hover:bg-muted rounded transition-colors"
                onClick={() => {
                  onAddReaction(message.id, emoji)
                  setShowReactions(false)
                }}
              >
                {emoji}
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
