import type { Metadata } from 'next'
import { Geist, Geist_Mono } from 'next/font/google'
import { ThemeProvider } from '@/components/theme-provider'
import './globals.css'

const geistSans = Geist({
  subsets: ['latin'],
  variable: '--font-geist-sans',
})

const geistMono = Geist_Mono({
  subsets: ['latin'],
  variable: '--font-geist-mono',
})

export const metadata: Metadata = {
  title: 'Messaging Platform | Real-time Chat & DMs',
  description: 'Advanced messaging platform with support for voice messages, file attachments, emoji reactions, and group chats',
  keywords: ['messaging', 'chat', 'dm', 'voice messages', 'real-time'],
  themeColor: [
    { media: '(prefers-color-scheme: light)', content: 'white' },
    { media: '(prefers-color-scheme: dark)', content: '#0a0a0a' },
  ],
}

export const viewport = {
  width: 'device-width',
  initialScale: 1,
  userScalable: false,
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={`${geistSans.variable} ${geistMono.variable} font-sans antialiased`}>
        <ThemeProvider>{children}</ThemeProvider>
      </body>
    </html>
  )
}
