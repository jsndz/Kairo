import type { Edit } from './operations'

export type OwnershipMap = Record<string, string> // clipId -> clientId

export type PermissionRequest = {
  reqId: string
  clipId: string
  fromClientId: string
  timeoutMs: number
  expiresAt: number
}

export type ConsensusProposal = {
  proposalId: string
  proposerId: string
  op: Edit
  description: string
  timeoutMs: number
  expiresAt: number
}

export type OwnershipSlice = {
  owners: OwnershipMap
  myClientId: string | null
  incomingPermission: PermissionRequest | null
  outgoingPermission: { reqId: string; clipId: string } | null
  activeConsensus: ConsensusProposal | null
  claimClips: (clipIds: string[]) => Promise<boolean>
  releaseClips: (clipIds: string[]) => void
  requestPermission: (clipId: string) => Promise<boolean>
  respondPermission: (reqId: string, approve: boolean) => void
  proposeConsensus: (op: Edit, description: string) => Promise<boolean>
  voteConsensus: (proposalId: string, approve: boolean) => void
}
