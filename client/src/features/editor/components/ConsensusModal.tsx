import { useEffect, useState } from 'react'
import { GitPullRequest, Check, X, AlertTriangle, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useEditor } from '../state'

export function ConsensusModal() {
  const proposal = useEditor(state => state.activeConsensus)
  const myClientId = useEditor(state => state.myClientId)
  const voteConsensus = useEditor(state => state.voteConsensus)
  const [remaining, setRemaining] = useState(30)

  useEffect(() => {
    if (!proposal) return
    const updateRemaining = () => {
      const left = Math.max(0, Math.ceil((proposal.expiresAt - Date.now()) / 1000))
      setRemaining(left)
      if (left <= 0) {
        voteConsensus(proposal.proposalId, false)
      }
    }
    updateRemaining()
    const timer = setInterval(updateRemaining, 500)
    return () => clearInterval(timer)
  }, [proposal, voteConsensus])

  if (!proposal) return null

  const isProposer = Boolean(myClientId && proposal.proposerId === myClientId)
  const description = proposal.description || proposal.op.type

  return (
    <div
      className="fixed inset-0 bg-black/60 backdrop-blur-xs flex items-center justify-center z-50 p-4"
      role="dialog"
      aria-modal="true"
    >
      <div className="bg-neutral-900 border border-neutral-700 rounded-xl p-6 max-w-md w-full shadow-2xl">
        <div className="flex items-start gap-4">
          <div className="p-3 bg-sky-500/15 text-sky-400 rounded-lg shrink-0">
            {isProposer ? <Loader2 size={24} className="animate-spin" /> : <GitPullRequest size={24} />}
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center justify-between">
              <h3 className="text-base font-semibold text-neutral-100">
                {isProposer ? 'Proposal Awaiting Peer' : 'Consensus Proposal'}
              </h3>
              <span className="text-xs font-mono text-sky-400 bg-sky-500/15 px-2 py-0.5 rounded">
                {remaining}s
              </span>
            </div>
            <p className="text-sm text-neutral-400 mt-1">
              {isProposer
                ? 'You proposed a project-wide structural change. Waiting for peer agreement.'
                : 'Peer proposed a project-wide structural change. Both editors must agree before it can take effect.'}
            </p>
          </div>
        </div>

        <div className="mt-4 p-3 bg-neutral-950 border border-neutral-800 rounded-lg flex items-center gap-3">
          <AlertTriangle size={16} className="text-amber-400 shrink-0" />
          <div className="min-w-0">
            <div className="text-xs font-medium text-neutral-500 uppercase tracking-wider">Proposed Mutation</div>
            <div className="text-sm font-semibold text-neutral-200 truncate">{description}</div>
          </div>
        </div>

        <div className="flex justify-end gap-3 mt-6">
          {isProposer ? (
            <Button
              variant="outline"
              size="sm"
              className="text-neutral-300 border-neutral-700 hover:bg-neutral-800"
              onClick={() => voteConsensus(proposal.proposalId, false)}
            >
              <X size={15} className="mr-1.5" /> Cancel Proposal
            </Button>
          ) : (
            <>
              <Button
                variant="ghost"
                size="sm"
                className="text-neutral-400 hover:text-neutral-100 hover:bg-neutral-800"
                onClick={() => voteConsensus(proposal.proposalId, false)}
              >
                <X size={15} className="mr-1.5" /> Reject
              </Button>
              <Button
                size="sm"
                className="bg-sky-600 hover:bg-sky-500 text-white font-medium"
                onClick={() => voteConsensus(proposal.proposalId, true)}
              >
                <Check size={15} className="mr-1.5" /> Approve ({remaining}s)
              </Button>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
