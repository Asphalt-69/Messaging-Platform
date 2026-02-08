'use client'

import { useState } from 'react'
import ChatList from '@/components/chat-list'
import ChatView from '@/components/chat-view'

export default function Home() {
  const [selectedChatId, setSelectedChatId] = useState<string | null>(null)
  const [showChatList, setShowChatList] = useState(true)

  const handleSelectChat = (chatId: string) => {
    setSelectedChatId(chatId)
    setShowChatList(false)
  }

  const handleBackClick = () => {
    setShowChatList(true)
  }

  return (
    <main className="h-screen flex bg-background overflow-hidden">
      {/* Chat List - Hidden on mobile when chat is selected */}
      <div
        className={`w-full md:w-80 h-full border-r border-border transition-all ${
          showChatList ? 'block' : 'hidden md:block'
        }`}
      >
        <ChatList selectedChatId={selectedChatId || undefined} onSelectChat={handleSelectChat} />
      </div>

      {/* Chat View - Full width on mobile, flex-1 on desktop */}
      <div
        className={`flex-1 h-full transition-all ${
          showChatList ? 'hidden md:flex' : 'flex'
        } flex-col`}
      >
        <ChatView chatId={selectedChatId || undefined} onBackClick={handleBackClick} />
      </div>
    </main>
  )
}
