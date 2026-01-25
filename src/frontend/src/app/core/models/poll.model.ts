export interface PollOption {
  id: string;
  label: string;
}

export type AccessStructureType = 'officials_only' | 'auditors_only' | 'both';

export interface Poll {
  id: string;
  title: string;
  description?: string;
  creator_id: string;
  options: PollOption[];
  status: PollStatus;
  start_time?: string;
  end_time?: string;
  ballot_commitment?: string;
  access_structure_type: AccessStructureType;
  min_auditors_required: number;
  min_officials_required: number;
  total_auditors: number;
  total_officials: number;
  created_at: string;
  updated_at: string;
}

export type PollStatus = 'draft' | 'open' | 'closed' | 'revealed';

export interface BallotResponse {
  id: string;
  poll_id: string;
  vote_commitment: string;
  cast_at: string;
}

export interface ShareDistribution {
  poll_id: string;
  group_name: string;
  share_index: number;
  share_value: string;
  commitment: string;
}

export interface ShareStatus {
  poll_id: string;
  auditor_shares: number;
  min_auditors_required: number;
  official_shares: number;
  min_officials_required: number;
  can_reveal: boolean;
  contributed_by: string[];
}

export interface CreatePollRequest {
  title: string;
  description?: string;
  options: PollOption[];
  access_structure_type: AccessStructureType;
  min_auditors_required: number;
  min_officials_required: number;
  start_time?: string;
  end_time?: string;
}

export interface PollResultResponse {
  poll_id: string;
  results: Record<string, number>;
  total_votes: number;
  revealed_at: string;
}

export const POLL_STATUS_LABELS: Record<PollStatus, string> = {
  draft: 'Draft',
  open: 'Open',
  closed: 'Closed',
  revealed: 'Revealed'
};

export const POLL_STATUS_COLORS: Record<PollStatus, string> = {
  draft: 'gray',
  open: 'green',
  closed: 'blue',
  revealed: 'purple'
};
