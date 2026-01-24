import { UserRole } from './auth.model';

export interface PollOption {
  id: string;
  label: string;
}

export interface Poll {
  id: string;
  title: string;
  description: string;
  creator_id: string;
  options: PollOption[];
  status: PollStatus;
  start_time?: string;
  end_time?: string;
  ballot_commitment?: string;
  created_at: string;
  updated_at: string;
}

export type PollStatus = 'draft' | 'open' | 'closed' | 'frozen' | 'revealed';

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
  auditor_threshold: number;
  official_shares: number;
  official_threshold: number;
  can_reveal: boolean;
  contributed_by: string[];
}

export interface CreatePollRequest {
  title: string;
  description: string;
  options: PollOption[];
  start_time?: string;
  end_time?: string;
  auditor_threshold: number;
  auditor_total: number;
  official_threshold: number;
  official_total: number;
}

export const POLL_STATUS_LABELS: Record<PollStatus, string> = {
  draft: 'Draft',
  open: 'Open',
  closed: 'Closed',
  frozen: 'Frozen',
  revealed: 'Revealed'
};

export const POLL_STATUS_COLORS: Record<PollStatus, string> = {
  draft: 'gray',
  open: 'green',
  closed: 'yellow',
  frozen: 'blue',
  revealed: 'purple'
};
