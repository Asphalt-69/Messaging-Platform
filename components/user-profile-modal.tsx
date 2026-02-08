'use client'

import { useState } from 'react'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Upload, X } from 'lucide-react'
import Image from 'next/image'

interface UserProfileModalProps {
  isOpen: boolean
  onClose: () => void
  isOwnProfile?: boolean
  user?: {
    id: string
    name: string
    username: string
    avatar_url: string | null
  }
}

export function UserProfileModal({
  isOpen,
  onClose,
  isOwnProfile = false,
  user,
}: UserProfileModalProps) {
  const [name, setName] = useState(user?.name || '')
  const [username, setUsername] = useState(user?.username || '')
  const [avatarUrl, setAvatarUrl] = useState(user?.avatar_url || '')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSave = async () => {
    if (!name.trim() || !username.trim()) {
      setError('Name and username are required')
      return
    }

    setIsLoading(true)
    setError('')

    try {
      const response = await fetch('/api/profile', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, username, avatar_url: avatarUrl }),
      })

      if (!response.ok) {
        throw new Error('Failed to update profile')
      }

      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred')
    } finally {
      setIsLoading(false)
    }
  }

  const handleAvatarChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      const reader = new FileReader()
      reader.onloadend = () => {
        setAvatarUrl(reader.result as string)
      }
      reader.readAsDataURL(file)
    }
  }

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isOwnProfile ? 'Edit Profile' : 'View Profile'}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          {/* Avatar */}
          <div className="flex justify-center">
            <div className="relative">
              <div className="h-32 w-32 rounded-full overflow-hidden bg-muted border-2 border-border flex items-center justify-center">
                {avatarUrl ? (
                  <Image
                    src={avatarUrl}
                    alt={name || 'User'}
                    fill
                    className="object-cover"
                  />
                ) : (
                  <div className="text-4xl text-muted-foreground">👤</div>
                )}
              </div>
              {isOwnProfile && (
                <label className="absolute bottom-0 right-0 bg-primary text-primary-foreground rounded-full p-2 cursor-pointer hover:bg-primary/90 transition">
                  <Upload className="h-4 w-4" />
                  <input
                    type="file"
                    accept="image/*"
                    onChange={handleAvatarChange}
                    className="hidden"
                  />
                </label>
              )}
            </div>
          </div>

          {/* Name */}
          <div>
            <label className="text-sm font-medium">Name</label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={!isOwnProfile}
              placeholder="Enter your name"
              className="mt-1"
            />
          </div>

          {/* Username */}
          <div>
            <label className="text-sm font-medium">Username</label>
            <Input
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              disabled={!isOwnProfile}
              placeholder="Enter your username"
              className="mt-1"
            />
          </div>

          {error && <div className="text-sm text-destructive">{error}</div>}

          {/* Actions */}
          {isOwnProfile && (
            <div className="flex gap-2 pt-4">
              <Button variant="outline" onClick={onClose} className="flex-1">
                Cancel
              </Button>
              <Button onClick={handleSave} disabled={isLoading} className="flex-1">
                {isLoading ? 'Saving...' : 'Save'}
              </Button>
            </div>
          )}

          {!isOwnProfile && (
            <Button onClick={onClose} className="w-full">
              Close
            </Button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
