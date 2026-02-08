'use client'

import { useState, useRef, useEffect } from 'react'
import {
  Send,
  Paperclip,
  Smile,
  Mic,
  X,
  Volume2,
  Square,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'

interface MessageInputProps {
  onSendMessage: (content: string, attachments: File[]) => void
  onSendVoiceMessage: (audioBlob: Blob, duration: number) => void
  replyingTo?: {
    id: string
    content: string
    senderName: string
  }
  onCancelReply?: () => void
}

export default function MessageInput({
  onSendMessage,
  onSendVoiceMessage,
  replyingTo,
  onCancelReply,
}: MessageInputProps) {
  const [message, setMessage] = useState('')
  const [attachments, setAttachments] = useState<File[]>([])
  const [isRecording, setIsRecording] = useState(false)
  const [recordingTime, setRecordingTime] = useState(0)
  const [showEmojiPicker, setShowEmojiPicker] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const chunksRef = useRef<Blob[]>([])
  const recordingIntervalRef = useRef<NodeJS.Timeout | null>(null)

  useEffect(() => {
    return () => {
      if (recordingIntervalRef.current) {
        clearInterval(recordingIntervalRef.current)
      }
    }
  }, [])

  const handleSend = () => {
    if (!message.trim() && attachments.length === 0) return

    onSendMessage(message, attachments)
    setMessage('')
    setAttachments([])
  }

  const handleAttachmentClick = () => {
    fileInputRef.current?.click()
  }

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || [])
    setAttachments((prev) => [...prev, ...files])
  }

  const removeAttachment = (index: number) => {
    setAttachments((prev) => prev.filter((_, i) => i !== index))
  }

  const startRecording = async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      mediaRecorderRef.current = new MediaRecorder(stream)
      chunksRef.current = []

      mediaRecorderRef.current.ondataavailable = (e) => {
        chunksRef.current.push(e.data)
      }

      mediaRecorderRef.current.onstop = () => {
        const blob = new Blob(chunksRef.current, { type: 'audio/webm' })
        onSendVoiceMessage(blob, recordingTime)
        setRecordingTime(0)
        stream.getTracks().forEach((track) => track.stop())
      }

      mediaRecorderRef.current.start()
      setIsRecording(true)

      recordingIntervalRef.current = setInterval(() => {
        setRecordingTime((prev) => prev + 1)
      }, 1000)
    } catch (error) {
      console.log('[v0] Error accessing microphone:', error)
    }
  }

  const stopRecording = () => {
    if (mediaRecorderRef.current && isRecording) {
      mediaRecorderRef.current.stop()
      setIsRecording(false)
      if (recordingIntervalRef.current) {
        clearInterval(recordingIntervalRef.current)
      }
    }
  }

  const addEmoji = (emoji: string) => {
    setMessage((prev) => prev + emoji)
    setShowEmojiPicker(false)
  }

  const commonEmojis = [
    '😀',
    '😂',
    '❤️',
    '👍',
    '🎉',
    '🚀',
    '✨',
    '👏',
    '😍',
    '🔥',
    '💯',
    '😎',
  ]

  return (
    <div className="border-t border-border bg-background p-4 space-y-3">
      {/* Reply Preview */}
      {replyingTo && (
        <div className="flex items-center justify-between bg-muted p-3 rounded-lg border-l-2 border-accent">
          <div className="flex-1 min-w-0">
            <p className="text-xs font-semibold text-muted-foreground">
              Replying to {replyingTo.senderName}
            </p>
            <p className="text-sm text-foreground truncate">
              {replyingTo.content}
            </p>
          </div>
          <Button
            size="icon"
            variant="ghost"
            className="h-6 w-6 flex-shrink-0"
            onClick={onCancelReply}
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      )}

      {/* Attachments Preview */}
      {attachments.length > 0 && (
        <div className="flex gap-2 flex-wrap">
          {attachments.map((file, index) => (
            <div
              key={index}
              className="relative group bg-muted p-2 rounded-lg flex items-center gap-2"
            >
              <span className="text-xs truncate max-w-xs">{file.name}</span>
              <button
                onClick={() => removeAttachment(index)}
                className="opacity-0 group-hover:opacity-100 transition-opacity"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Recording Status */}
      {isRecording && (
        <div className="flex items-center justify-between bg-destructive/10 border border-destructive text-destructive p-3 rounded-lg">
          <div className="flex items-center gap-2">
            <div className="w-2 h-2 rounded-full bg-destructive animate-pulse" />
            <span className="text-sm font-medium">
              Recording... {recordingTime}s
            </span>
          </div>
          <Button
            size="sm"
            variant="ghost"
            onClick={stopRecording}
            className="text-destructive hover:text-destructive"
          >
            <Square className="h-4 w-4 mr-2" />
            Stop
          </Button>
        </div>
      )}

      {/* Emoji Picker */}
      {showEmojiPicker && (
        <div className="grid grid-cols-6 gap-2 bg-muted p-3 rounded-lg">
          {commonEmojis.map((emoji) => (
            <button
              key={emoji}
              className="text-2xl hover:bg-background rounded p-1 transition-colors"
              onClick={() => addEmoji(emoji)}
            >
              {emoji}
            </button>
          ))}
        </div>
      )}

      {/* Input Area */}
      <div className="flex items-end gap-2">
        {/* Attachment Button */}
        <Button
          size="icon"
          variant="ghost"
          onClick={handleAttachmentClick}
          className="flex-shrink-0"
        >
          <Paperclip className="h-5 w-5" />
        </Button>

        {/* Emoji Button */}
        <Button
          size="icon"
          variant="ghost"
          onClick={() => setShowEmojiPicker(!showEmojiPicker)}
          className="flex-shrink-0"
        >
          <Smile className="h-5 w-5" />
        </Button>

        {/* Text Input */}
        <Textarea
          placeholder="Type a message..."
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault()
              handleSend()
            }
          }}
          className="flex-1 resize-none max-h-24"
          rows={1}
        />

        {/* Voice or Send Button */}
        {message.trim() || attachments.length > 0 ? (
          <Button
            size="icon"
            onClick={handleSend}
            className="flex-shrink-0 bg-accent hover:bg-accent/90"
          >
            <Send className="h-5 w-5" />
          </Button>
        ) : (
          <Button
            size="icon"
            variant={isRecording ? 'destructive' : 'default'}
            onClick={isRecording ? stopRecording : startRecording}
            className="flex-shrink-0"
          >
            {isRecording ? (
              <Square className="h-5 w-5" />
            ) : (
              <Mic className="h-5 w-5" />
            )}
          </Button>
        )}

        {/* Hidden File Input */}
        <input
          ref={fileInputRef}
          type="file"
          multiple
          onChange={handleFileSelect}
          className="hidden"
          accept="image/*,.pdf,.doc,.docx,.txt"
        />
      </div>
    </div>
  )
}
