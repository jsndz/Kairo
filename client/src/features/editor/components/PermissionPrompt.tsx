import { useEffect, useState } from 'react'
import { ShieldAlert, Check, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useEditor } from '../state'

export function PermissionPrompt() {
  const incoming = useEditor(state => state.incomingPermission)
  const project = useEditor(state => state.project)
  const respondPermission = useEditor(state => state.respondPermission)
  const [remaining, setRemaining] = useState(30)

  useEffect(() => {
    if (!incoming) return
    const updateRemaining = () => {
      const left = Math.max(0, Math.ceil((incoming.expiresAt - Date.now()) / 1000))
      setRemaining(left)
      if (left <= 0) {
        respondPermission(incoming.reqId, false)
      }
    }
    updateRemaining()
    const timer = setInterval(updateRemaining, 500)
    return () => clearInterval(timer)
  }, [incoming, respondPermission])

  if (!incoming) return null

  const targetClip = project.clips.find(c => c.id === incoming.clipId)
  const assetName = targetClip ? project.assets[targetClip.mediaId]?.name || 'Untitled Clip' : 'Clip'

  return (
    <div className="permission-prompt-toast fixed bottom-4 right-4 z-50 flex flex-col gap-3 p-4 bg-neutral-900 border border-amber-500/60 rounded-xl shadow-2xl max-w-sm">
      <div className="flex items-start gap-3">
        <div className="p-2 bg-amber-500/15 text-amber-400 rounded-lg shrink-0">
          <ShieldAlert size={20} />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between">
            <h4 className="text-sm font-semibold text-neutral-100">Permission Request</h4>
            <span className="text-xs font-mono text-amber-400 bg-amber-500/15 px-1.5 py-0.5 rounded">
              {remaining}s
            </span>
          </div>
          <p className="text-xs text-neutral-300 mt-1 leading-relaxed">
            Peer requested permission to edit <strong className="text-neutral-100">{assetName}</strong>.
          </p>
        </div>
      </div>
      <div className="flex justify-end gap-2 pt-2 border-t border-neutral-800">
        <Button
          variant="ghost"
          size="sm"
          className="h-8 px-3 text-xs text-neutral-400 hover:text-neutral-100 hover:bg-neutral-800"
          onClick={() => respondPermission(incoming.reqId, false)}
        >
          <X size={14} className="mr-1" /> Deny
        </Button>
        <Button
          size="sm"
          className="h-8 px-3 text-xs bg-emerald-600 hover:bg-emerald-500 text-white font-medium"
          onClick={() => respondPermission(incoming.reqId, true)}
        >
          <Check size={14} className="mr-1" /> Approve ({remaining}s)
        </Button>
      </div>
    </div>
  )
}
